package producer

import (
	kafka "github.com/Shopify/sarama"
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"sync"
	"time"
)

const (
	partRandom     = "Random"
	partRoundrobin = "Roundrobin"
	partHash       = "Hash"
	compressNone   = "None"
	compressGZIP   = "Zip"
	compressSnappy = "Snappy"
)

// Kafka producer plugin
// Configuration example
//
// - "producer.Kafka":
//   Enable: true
//   Partitioner: "Roundrobin"
//   RequiredAcks: 0
//   TimeoutMs: 0
//   Compression: "Snappy"
//   BatchMinCount: 10
//   BatchSizeByte: 16384
//   BatchTimeoutSec: 5
//   BufferSizeMaxKB: 524288
//   BatchMaxCount: 0
//   ElectTimeoutMs: 1000
//   MetadataRefreshSec: 30
//   Servers:
//   	- "192.168.222.30:9092"
//   Stream:
//     - "console"
//     - "_GOLLUM_"
//   Topic:
//     "console" : "default"
//     "_GOLLUM_"  : "default"
//
// Partitioner sets the distribution algorithm to use. Valid values (case
// sensitive) are: "Random","Roundrobin","Hash". By default "Hash" is set.
//
// RequiredAcks defines the acknowledgement level required by the broker.
// 0 = No responses required. 1 = wait for the local commit. -1 = wait for
// all replicas to commit. >1 = wait for a specific number of commits.
// By default this is set to 1.
//
// TimeoutMs denotes the maximum time the broker will wait for acks. This
// setting becomes active when RequiredAcks is set to wait for multiple commits.
// By default this is set to 1500.
//
// Compression sets the method of compression to use. Valid values (case
// sensitive) are: "None","Zip","Snappy". By default "None" is set.
//
// BufferSizeMaxKB defines the maximum allowed message size. By default this is
// set to 1 MB.
//
// BatchSizeByte sets the mimimum number of bytes to collect before a new flush
// is triggered. By default this is set to 8192.
//
// BatchMinCount sets the minimum number of messages required to trigger a
// flush. By default this is set to 1.
//
// BatchMaxCount defines the maximum number of messages processed per
// request. By default this is set to 0 for "unlimited".
//
// BatchTimeoutSec sets the minimum time in seconds to pass after wich a new
// flush will be triggered. By default this is set to 3.
//
// ElectTimeoutMs defines the number of milliseconds to wait for the cluster to
// elect a new leader. Defaults to 250.
//
// MetadataRefreshSec set the interval in seconds for fetching cluster metadata.
// By default this is set to 10.
//
// Topic maps a stream to a specific kafka topic. You can define the
// wildcard stream (*) here, too. All streams that do not have a specific
// mapping will go to this stream (including _GOLLUM_).
// If no topic mappings are set all messages will be send to "default".
type Kafka struct {
	standardProducer
	servers        []string
	topic          map[shared.MessageStreamID]string
	defaultTopic   string
	client         *kafka.Client
	clientConfig   *kafka.ClientConfig
	producer       *kafka.Producer
	producerConfig *kafka.ProducerConfig
}

func init() {
	shared.RuntimeType.Register(Kafka{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kafka) Configure(conf shared.PluginConfig) error {
	// If not defined, delimiter is not used (override default value)
	if !conf.HasValue("Delimiter") {
		conf.Override("Delimiter", "")
	}

	err := prod.standardProducer.Configure(conf)
	if err != nil {
		return err
	}

	prod.clientConfig = kafka.NewClientConfig()
	prod.producerConfig = kafka.NewProducerConfig()
	prod.servers = conf.GetStringArray("Servers", []string{})
	prod.topic = make(map[shared.MessageStreamID]string)
	prod.defaultTopic = "default"

	defaultMapping := make(map[string]string)
	defaultMapping[shared.WildcardStream] = prod.defaultTopic

	topicMap := conf.GetStringMap("Topic", defaultMapping)
	for stream, topic := range topicMap {
		prod.topic[shared.GetStreamID(stream)] = topic
	}

	wildcardTopic, wildcardTopicSet := prod.topic[shared.WildcardStreamID]
	if wildcardTopicSet {
		prod.defaultTopic = wildcardTopic
	}

	switch conf.GetString("Partitioner", partRandom) {
	case partRandom:
		prod.producerConfig.Partitioner = kafka.NewRandomPartitioner
	case partRoundrobin:
		prod.producerConfig.Partitioner = kafka.NewRoundRobinPartitioner
	default:
		// Don't set == partHash
	case partHash:
		prod.producerConfig.Partitioner = kafka.NewHashPartitioner
	}

	prod.producerConfig.RequiredAcks = kafka.RequiredAcks(conf.GetInt("RequiredAcks", int(kafka.WaitForLocal)))
	prod.producerConfig.Timeout = time.Duration(conf.GetInt("TimoutMs", 1500)) * time.Millisecond

	switch conf.GetString("Compression", compressNone) {
	default:
		// Don't set == compressNome
	case compressNone:
		prod.producerConfig.Compression = kafka.CompressionNone
	case compressGZIP:
		prod.producerConfig.Compression = kafka.CompressionGZIP
	case compressSnappy:
		prod.producerConfig.Compression = kafka.CompressionSnappy
	}

	prod.producerConfig.FlushMsgCount = conf.GetInt("BatchMinCount", 1)
	prod.producerConfig.FlushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second
	prod.producerConfig.FlushByteCount = conf.GetInt("BatchSizeByte", 8192)

	prod.producerConfig.MaxMessageBytes = conf.GetInt("BufferSizeMaxKB", 1<<10) << 10
	prod.producerConfig.MaxMessagesPerReq = conf.GetInt("BatchMaxCount", 0)
	prod.producerConfig.RetryBackoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond

	prod.clientConfig.WaitForElection = prod.producerConfig.RetryBackoff
	prod.clientConfig.BackgroundRefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshSec", 10)) * time.Second

	return nil
}

func (prod *Kafka) send(msg shared.Message) {
	var err error

	// If we have not yet connected or the connection dropped: connect.
	if prod.client == nil || prod.client.Closed() {
		if prod.producer != nil {
			prod.producer.Close()
			prod.producer = nil
		}

		prod.client, err = kafka.NewClient("gollum", prod.servers, prod.clientConfig)
		if err != nil {
			Log.Error.Print("Kafka client error:", err)
			return
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		prod.producer, err = kafka.NewProducer(prod.client, prod.producerConfig)
		if err != nil {
			Log.Error.Print("Kafka producer error:", err)
			return
		}
	}

	if prod.client != nil && prod.producer != nil {
		// Send message
		topic, topicMapped := prod.topic[msg.PinnedStream]
		if !topicMapped {
			topic = prod.defaultTopic
		}

		prod.producer.Input() <- &kafka.MessageToSend{
			Topic: topic,
			Key:   nil,
			Value: kafka.StringEncoder(prod.format.ToString(msg)),
		}

		// Check for errors
		select {
		case err := <-prod.producer.Errors():
			Log.Error.Print("Kafka message error:", err)
		default:
		}
	}
}

// Produce writes to a buffer that is sent to a given socket.
func (prod Kafka) Produce(threads *sync.WaitGroup) {
	defer func() {
		if prod.producer != nil {
			prod.producer.Close()
		}
		if prod.client != nil && !prod.client.Closed() {
			prod.client.Close()
		}
		prod.markAsDone()
	}()

	prod.defaultControlLoop(threads, prod.send)
}
