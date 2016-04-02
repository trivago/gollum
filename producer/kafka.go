// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package producer

import (
	kafka "github.com/shopify/sarama"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	partRandom     = "random"
	partRoundrobin = "roundrobin"
	partHash       = "hash"
	compressNone   = "none"
	compressGZIP   = "zip"
	compressSnappy = "snappy"
)

// Kafka producer plugin
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the sarama library so most settings relate to that library.
// This producer uses a fuse breaker if any connection reports an error.
// Configuration example
//
//  - "producer.Kafka":
//    ClientId: "weblog"
//    Partitioner: "Roundrobin"
//    RequiredAcks: 1
//    TimeoutMs: 1500
//    SendRetries: 3
//    Compression: "None"
//    MaxOpenRequests: 5
//    MessageBufferCount: 256
//    BatchMinCount: 10
//    BatchMaxCount: 1
//    BatchSizeByte: 8192
//    BatchSizeMaxKB: 1024
//    BatchTimeoutMs: 3000
//    ServerTimeoutSec: 30
//    SendTimeoutMs: 250
//    ElectRetries: 3
//    ElectTimeoutMs: 250
//    MetadataRefreshMs: 10000
//    Servers:
//    	- "localhost:9092"
//    Topic:
//      "console" : "console"
//
// ClientId sets the client id of this producer. By default this is "gollum".
//
// Partitioner sets the distribution algorithm to use. Valid values are:
// "Random","Roundrobin" and "Hash". By default "Roundrobin" is set.
//
// KeyFormatter can define a formatter that extracts the key for a kafka message
// from the message payload. By default this is an empty string, which disables
// this feature. A good formatter for this can be format.Identifier.
//
// RequiredAcks defines the acknowledgment level required by the broker.
// 0 = No responses required. 1 = wait for the local commit. -1 = wait for
// all replicas to commit. >1 = wait for a specific number of commits.
// By default this is set to 1.
//
// TimeoutMs denotes the maximum time the broker will wait for acks. This
// setting becomes active when RequiredAcks is set to wait for multiple commits.
// By default this is set to 1500.
//
// SendRetries defines how many times to retry sending data before marking a
// server as not reachable. By default this is set to 0.
//
// Compression sets the method of compression to use. Valid values are:
// "None","Zip" and "Snappy". By default "None" is set.
//
// MaxOpenRequests defines the number of simultanious connections are allowed.
// By default this is set to 5.
//
// BatchMinCount sets the minimum number of messages required to trigger a
// flush. By default this is set to 1.
//
// BatchMaxCount defines the maximum number of messages processed per
// request. By default this is set to 0 for "unlimited".
//
// BatchSizeByte sets the minimum number of bytes to collect before a new flush
// is triggered. By default this is set to 8192.
//
// BatchSizeMaxKB defines the maximum allowed message size. By default this is
// set to 1024.
//
// BatchTimeoutMs sets the minimum time in milliseconds to pass after wich a new
// flush will be triggered. By default this is set to 3.
//
// MessageBufferCount sets the internal channel size for the kafka client.
// By default this is set to 256.
//
// ServerTimeoutSec defines the time after which a connection is set to timed
// out. By default this is set to 30 seconds.
//
// SendTimeoutMs defines the number of milliseconds to wait for a server to
// resond before triggering a timeout. Defaults to 250.
//
// ElectRetries defines how many times to retry during a leader election.
// By default this is set to 3.
//
// ElectTimeoutMs defines the number of milliseconds to wait for the cluster to
// elect a new leader. Defaults to 250.
// GracePeriodMs defines the number of milliseconds to wait for Sarama to
//
// accept a single message. After this period a message is dropped.
// By default this is set to 5ms.
//
// MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
// By default this is set to 10000. This corresponds to the JVM setting
// `topic.metadata.refresh.interval.ms`.
//
// Servers contains the list of all kafka servers to connect to.  By default this
// is set to contain only "localhost:9092".
//
// Topic maps a stream to a specific kafka topic. You can define the
// wildcard stream (*) here, too. If defined, all streams that do not have a
// specific mapping will go to this topic (including _GOLLUM_).
// If no topic mappings are set the stream names will be used as topic.
type Kafka struct {
	core.ProducerBase
	servers          []string
	topicGuard       *sync.RWMutex
	topic            map[core.MessageStreamID]string
	clientID         string
	client           kafka.Client
	config           *kafka.Config
	producer         kafka.AsyncProducer
	counters         map[string]*int64
	missCount        int64
	lastMetricUpdate time.Time
	gracePeriod      time.Duration
	keyFormat        core.Formatter
}

const (
	kafkaMetricMessages     = "Kafka:Messages-"
	kafkaMetricMessagesSec  = "Kafka:MessagesSec-"
	kafkaMetricUnresponsive = "Kafka:Unresponsive-"
)

func init() {
	core.TypeRegistry.Register(Kafka{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kafka) Configure(conf core.PluginConfigReader) error {
	prod.ProducerBase.Configure(conf)
	prod.SetStopCallback(prod.close)
	prod.SetCheckFuseCallback(prod.checkAllTopics)

	kafka.Logger = prod.Log.Note

	if conf.HasValue("KeyFormatter") {
		keyFormatter := conf.GetPlugin("KeyFormatter", "format.Identifier", tcontainer.NewMarshalMap())
		prod.keyFormat = keyFormatter.(core.Formatter)
	} else {
		prod.keyFormat = nil
	}

	prod.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.lastMetricUpdate = time.Now()
	prod.gracePeriod = time.Duration(conf.GetInt("GracePeriodMs", 5)) * time.Millisecond
	prod.topicGuard = new(sync.RWMutex)
	prod.topic = conf.GetStreamMap("Topic", "")
	prod.counters = make(map[string]*int64)

	prod.config = kafka.NewConfig()
	prod.config.ClientID = conf.GetString("ClientId", "gollum")
	prod.config.ChannelBufferSize = conf.GetInt("MessageBufferCount", 256)

	prod.config.Net.MaxOpenRequests = conf.GetInt("MaxOpenRequests", 5)
	prod.config.Net.DialTimeout = time.Duration(conf.GetInt("ServerTimeoutSec", 30)) * time.Second
	prod.config.Net.ReadTimeout = prod.config.Net.DialTimeout
	prod.config.Net.WriteTimeout = prod.config.Net.DialTimeout

	prod.config.Metadata.Retry.Max = conf.GetInt("ElectRetries", 3)
	prod.config.Metadata.Retry.Backoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	prod.config.Metadata.RefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshMs", 10000)) * time.Millisecond

	prod.config.Producer.MaxMessageBytes = conf.GetInt("BatchSizeMaxKB", 1<<10) << 10
	prod.config.Producer.RequiredAcks = kafka.RequiredAcks(conf.GetInt("RequiredAcks", int(kafka.WaitForLocal)))
	prod.config.Producer.Timeout = time.Duration(conf.GetInt("TimoutMs", 1500)) * time.Millisecond
	prod.config.Producer.Flush.Bytes = conf.GetInt("BatchSizeByte", 8192)
	prod.config.Producer.Flush.Messages = conf.GetInt("BatchMinCount", 1)
	prod.config.Producer.Flush.Frequency = time.Duration(conf.GetInt("BatchTimeoutMs", 3000)) * time.Millisecond
	prod.config.Producer.Flush.MaxMessages = conf.GetInt("BatchMaxCount", 0)
	prod.config.Producer.Retry.Max = conf.GetInt("SendRetries", 0)
	prod.config.Producer.Retry.Backoff = time.Duration(conf.GetInt("SendTimeoutMs", 100)) * time.Millisecond

	prod.config.Producer.Return.Errors = true

	switch strings.ToLower(conf.GetString("Compression", compressNone)) {
	default:
		fallthrough
	case compressNone:
		prod.config.Producer.Compression = kafka.CompressionNone
	case compressGZIP:
		prod.config.Producer.Compression = kafka.CompressionGZIP
	case compressSnappy:
		prod.config.Producer.Compression = kafka.CompressionSnappy
	}

	switch strings.ToLower(conf.GetString("Partitioner", partRoundrobin)) {
	case partRandom:
		prod.config.Producer.Partitioner = kafka.NewRandomPartitioner
	case partRoundrobin:
		prod.config.Producer.Partitioner = kafka.NewRoundRobinPartitioner
	default:
		fallthrough
	case partHash:
		prod.config.Producer.Partitioner = kafka.NewHashPartitioner
	}

	for _, topic := range prod.topic {
		tgo.Metric.New(kafkaMetricMessages + topic)
		tgo.Metric.New(kafkaMetricMessagesSec + topic)
		tgo.Metric.New(kafkaMetricUnresponsive + topic)
		prod.counters[topic] = new(int64)
	}

	return conf.Errors.OrNil()
}

func (prod *Kafka) pollResults() {
	// Check for results
	keepPolling := true
	timeout := time.NewTimer(prod.config.Producer.Flush.Frequency / 2)
	for keepPolling {
		select {
		case err := <-prod.producer.Errors():
			if msg, hasMsg := err.Msg.Metadata.(*core.Message); hasMsg {
				prod.Drop(msg)
			}

		case <-timeout.C:
			keepPolling = false
		}
	}

	// Update metrics
	for topic, counter := range prod.counters {
		duration := time.Since(prod.lastMetricUpdate)
		count := atomic.SwapInt64(counter, 0)
		countPerSec := float64(count)/duration.Seconds() + 0.5

		tgo.Metric.Add(kafkaMetricMessages+topic, count)
		tgo.Metric.SetF(kafkaMetricMessagesSec+topic, countPerSec)
	}

	prod.lastMetricUpdate = time.Now()
}

func (prod *Kafka) registerNewTopic(streamID core.MessageStreamID) string {
	prod.topicGuard.Lock()
	defer prod.topicGuard.Unlock()

	// Check again to avoid races
	topic, topicMapped := prod.topic[streamID]
	if !topicMapped {
		// Use wildcard fallback or stream name if not set
		topic, topicMapped = prod.topic[core.WildcardStreamID]
		if !topicMapped {
			topic = core.StreamRegistry.GetStreamName(streamID)
		}

		tgo.Metric.New(kafkaMetricMessages + topic)
		tgo.Metric.New(kafkaMetricMessagesSec + topic)
		tgo.Metric.New(kafkaMetricUnresponsive + topic)
		prod.counters[topic] = new(int64)
		prod.topic[streamID] = topic
	}

	return topic
}

func (prod *Kafka) produceMessage(msg *core.Message) {
	originalMsg := *msg
	prod.ProducerBase.Format(msg)

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	topic, topicMapped := prod.topic[msg.StreamID]
	prod.topicGuard.RUnlock()
	if !topicMapped {
		topic = prod.registerNewTopic(msg.StreamID)
		prod.topicGuard.RLock()
	}

	if isConnected, err := prod.isConnected(topic); !isConnected {
		prod.Drop(msg)
		if err != nil {
			prod.Log.Error.Printf("%s is not connected: %s", topic, err.Error())
		}
		prod.Control() <- core.PluginControlFuseBurn
		return // ### return, not connected ###
	}

	kafkaMsg := &kafka.ProducerMessage{
		Topic:    topic,
		Value:    kafka.ByteEncoder(msg.Data),
		Metadata: &originalMsg,
	}

	if prod.keyFormat != nil {
		keyMsg := *msg
		prod.keyFormat.Format(&keyMsg)
		kafkaMsg.Key = kafka.ByteEncoder(keyMsg.Data)
	}

	// Sarama can block on single messages if all buffers are full.
	// So we stop trying after a few milliseconds
	timeout := time.NewTimer(prod.gracePeriod)
	select {
	case prod.producer.Input() <- kafkaMsg:
		timeout.Stop()
		atomic.AddInt64(prod.counters[topic], 1)

	case <-timeout.C:
		// Sarama channels are full -> drop
		prod.Drop(&originalMsg)
		tgo.Metric.Inc(kafkaMetricUnresponsive + topic)
	}
}

func (prod *Kafka) checkAllTopics() bool {
	topics, err := prod.client.Topics()
	if err != nil {
		prod.Log.Error.Print("Failed to get topics ", err.Error())
	}

	for _, topic := range topics {
		connected, _ := prod.isConnected(topic)
		if !connected {
			return false
		}
	}

	return true
}

func (prod *Kafka) isConnected(topic string) (bool, error) {
	if prod.client == nil || prod.producer == nil {
		if !prod.tryOpenConnection() {
			return false, nil // ### return, error ###
		}
	}

	partitions, err := prod.client.Partitions(topic)
	if err != nil {
		return false, err // ### return, error ###
	}

	for _, p := range partitions {
		broker, err := prod.client.Leader(topic, p)
		if err != nil {
			return false, err // ### return, error ###
		}

		// TODO: this function only returns false if the connection has explicitly
		//       been closed [Sarama 1.8.0]!
		return broker.Connected()
	}

	return true, nil
}

func (prod *Kafka) tryOpenConnection() bool {
	// Reconnect the client first
	if prod.client == nil {
		if client, err := kafka.NewClient(prod.servers, prod.config); err == nil {
			prod.client = client
		} else {
			prod.Log.Error.Print("Kafka client initialization error:", err)
			return false // ### return, connection failed ###
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		if producer, err := kafka.NewAsyncProducerFromClient(prod.client); err == nil {
			prod.producer = producer
		} else {
			prod.Log.Error.Print("Kafka producer initialization error:", err)
			return false // ### return, connection failed ###
		}
	}

	return true
}

func (prod *Kafka) closeConnection() {
	prod.producer.Close()
	prod.client.Close()
}

func (prod *Kafka) close() {
	defer prod.WorkerDone()
	prod.closeConnection()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *Kafka) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.tryOpenConnection()
	prod.TickerMessageControlLoop(prod.produceMessage, prod.config.Producer.Flush.Frequency, prod.pollResults)
}
