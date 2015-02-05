package producer

import (
	"github.com/Shopify/sarama"
	"github.com/trivago/gollum/shared"
	"sync"
)

// Kafka producer plugin
// Configuration example
//
// - "producer.Kafka":
//   Enable: true
//   Servers:
//   	- "192.168.222.30:9092"
//   Stream:
//     - "console"
//     - "_GOLLUM_"
//   Topic:
//     "console" : "default"
//     "_GOLLUM_"  : "default"
type Kafka struct {
	standardProducer
	servers        []string
	topic          map[shared.MessageStreamID]string
	defaultTopic   string
	client         *sarama.Client
	clientConfig   *sarama.ClientConfig
	producer       *sarama.Producer
	producerConfig *sarama.ProducerConfig
}

func init() {
	shared.Plugin.Register(Kafka{})
}

// Create creates a new producer based on the current file producer.
func (prod Kafka) Create(conf shared.PluginConfig) (shared.Producer, error) {
	// If not defined, delimiter is not used (override default value)
	if !conf.HasValue("Delimiter") {
		conf.Override("Delimiter", "")
	}

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	prod.clientConfig = sarama.NewClientConfig()
	prod.producerConfig = sarama.NewProducerConfig()
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

	return prod, nil
}

func (prod *Kafka) send(msg shared.Message) {
	var err error

	// If we have not yet connected or the connection dropped: connect.
	if prod.client == nil || prod.client.Closed() {
		if prod.producer != nil {
			prod.producer.Close()
			prod.producer = nil
		}

		prod.client, err = sarama.NewClient("gollum", prod.servers, prod.clientConfig)
		if err != nil {
			shared.Log.Error("Kafka client error:", err)
			return
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		prod.producer, err = sarama.NewProducer(prod.client, prod.producerConfig)
		if err != nil {
			shared.Log.Error("Kafka producer error:", err)
			return
		}
	}

	if prod.client != nil && prod.producer != nil {
		// Send message
		topic, topicMapped := prod.topic[msg.PinnedStream]
		if !topicMapped {
			topic = prod.defaultTopic
		}

		prod.producer.Input() <- &sarama.MessageToSend{
			Topic: topic,
			Key:   nil,
			Value: sarama.StringEncoder(prod.format.ToString(msg)),
		}

		// Check for errors
		select {
		case err := <-prod.producer.Errors():
			shared.Log.Error("Kafka message error:", err)
		default:
		}
	}
}

// Produce writes to a buffer that is sent to a given socket.
func (prod Kafka) Produce(threads *sync.WaitGroup) {
	threads.Add(1)

	defer func() {
		if prod.producer != nil {
			prod.producer.Close()
		}
		if prod.client != nil && !prod.client.Closed() {
			prod.client.Close()
		}
		threads.Done()
	}()

	for {
		select {
		case message := <-prod.messages:
			prod.send(message)

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		}
	}
}
