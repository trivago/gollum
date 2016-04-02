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

package native

import (
	kafka "github.com/trivago/gollum/contrib/native/librdkafka"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// KafkaProducer librdkafka producer plugin
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the native librdkafka library so most settings relate to that.
// library. This producer does not use a fuse breaker.
// Configuration example
//
//  - "native.KafkaProducer":
//    ClientId: "weblog"
//    RequiredAcks: 1
//    TimeoutMs: 1500
//    SendRetries: 0
//    Compression: "none"
//    BatchSizeMaxKB: 1024
//    BatchMaxMessages: 100000
//    BatchMinMessages: 10000
//    BatchTimeoutMs: 1000
//    ServerTimeoutSec: 60
//    ServerMaxFails: 3
//    MetadataTimeoutSec: 60
//    MetadataRefreshMs: 300000
//    KeyFormatter: ""
//    Servers:
//    	- "localhost:9092"
//    Topic:
//      "console" : "console"
//
// SendRetries is mapped to message.send.max.retries.
//
// Compression is mapped to compression.codec. Please not that "zip" has to be
// used instead of "gzip".
//
// BatchSizeMaxKB is mapped to message.max.bytes.
//
// BatchMaxMessages is mapped to queue.buffering.max.messages.
//
// BatchMinMessages is mapped to batch.num.messages.
//
// BatchTimeoutMs is mapped to queue.buffering.max.ms.
//
// ServerTimeoutSec is mapped to socket.timeout.ms.
//
// ServerMaxFails is mapped to socket.max.fails.
//
// MetadataTimeoutSec is mapped to metadata.request.timeout.ms.
//
// MetadataRefreshMs is mapped to topic.metadata.refresh.interval.ms.
//
// Servers defines the list of brokers to produce messages to.
//
// Topic defines a stream to topic mapping.
//
// KeyFormatter defines the formatter used to extract keys from a message.
// Set to "" by default (disable).
type KafkaProducer struct {
	core.ProducerBase
	servers           []string
	clientID          string
	counters          map[string]*int64
	lastMetricUpdate  time.Time
	keyFormat         core.Formatter
	client            *kafka.Client
	config            kafka.Config
	topicRequiredAcks int
	topicTimeoutMs    int
	pollInterval      time.Duration
	streamToTopic     map[core.MessageStreamID]string
	topic             map[core.MessageStreamID]*kafka.Topic
	topicGuard        *sync.RWMutex
}

type messageWrapper struct {
	key   []byte
	value []byte
	user  []byte
}

const (
	kafkaMetricMessages         = "Kafka:Messages-"
	kafkaMetricMessagesSec      = "Kafka:MessagesSec-"
	kafkaMetricMessagesInflight = "Kafka:Inflight"
)

const (
	compressNone   = "none"
	compressGZIP   = "zip"
	compressSnappy = "snappy"
)

func init() {
	core.TypeRegistry.Register(KafkaProducer{})
}

func (m *messageWrapper) GetKey() []byte {
	return m.key
}

func (m *messageWrapper) GetPayload() []byte {
	return m.value
}

func (m *messageWrapper) GetUserdata() []byte {
	return m.user
}

// Configure initializes this producer with values from a plugin config.
func (prod *KafkaProducer) Configure(conf core.PluginConfigReader) error {
	prod.ProducerBase.Configure(conf)

	prod.SetStopCallback(prod.close)
	kafka.Log = prod.Log.Error

	if conf.HasValue("KeyFormatter") {
		keyFormatter := conf.GetPlugin("KeyFormatter", "format.Identifier", tcontainer.NewMarshalMap())
		prod.keyFormat = keyFormatter.(core.Formatter)
	} else {
		prod.keyFormat = nil
	}

	prod.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.lastMetricUpdate = time.Now()

	prod.counters = make(map[string]*int64)
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")
	prod.topic = make(map[core.MessageStreamID]*kafka.Topic)
	prod.topicRequiredAcks = conf.GetInt("RequiredAcks", 1)
	prod.topicTimeoutMs = conf.GetInt("TimeoutMs", 1)
	prod.topicGuard = new(sync.RWMutex)

	batchIntervalMs := conf.GetInt("BatchTimeoutMs", 1000)
	prod.pollInterval = time.Millisecond * time.Duration(batchIntervalMs)

	// Init librdkafka
	prod.config = kafka.NewConfig()

	/*kafkaVer := conf.GetString("ProtocolVersion", "0.8.2")
	verParts := strings.Split(kafkaVer, ".")
	multiplicator := 1000000
	verNumber := 0
	for _, n := range verParts {
		part, _ := strconv.Atoi(n)
		verNumber += part * multiplicator
		multiplicator /= 100
	}*/

	prod.config.Set("client.id", conf.GetString("ClientId", "gollum"))
	prod.config.Set("metadata.broker.list", strings.Join(prod.servers, ","))
	prod.config.SetI("message.max.bytes", conf.GetInt("BatchSizeMaxKB", 1<<10)<<10)
	prod.config.SetI("metadata.request.timeout.ms", int(conf.GetInt("MetadataTimeoutSec", 60)*1000))
	prod.config.SetI("topic.metadata.refresh.interval.ms", int(conf.GetInt("MetadataRefreshMs", 300000)))
	prod.config.SetI("socket.max.fails", int(conf.GetInt("ServerMaxFails", 3)))
	prod.config.SetI("socket.timeout.ms", int(conf.GetInt("ServerTimeoutSec", 60)*1000))
	prod.config.SetB("socket.keepalive.enable", true)
	prod.config.SetI("message.send.max.retries", conf.GetInt("SendRetries", 0))
	prod.config.SetI("queue.buffering.max.messages", conf.GetInt("BatchMaxMessages", 100000))
	prod.config.SetI("queue.buffering.max.ms", batchIntervalMs)
	prod.config.SetI("batch.num.messages", conf.GetInt("BatchMinMessages", 10000))
	//prod.config.SetI("protocol.version", verNumber)

	switch strings.ToLower(conf.GetString("Compression", compressNone)) {
	default:
		prod.config.Set("compression.codec", "none")
	case compressGZIP:
		prod.config.Set("compression.codec", "gzip")
	case compressSnappy:
		prod.config.Set("compression.codec", "snappy")
	}

	tgo.Metric.New(kafkaMetricMessagesInflight)
	return conf.Errors.OrNil()
}

func (prod *KafkaProducer) registerNewTopic(streamID core.MessageStreamID) *kafka.Topic {
	prod.topicGuard.Lock()
	defer prod.topicGuard.Unlock()

	topic, topicMapped := prod.topic[streamID]
	if !topicMapped {
		topicName, isMapped := prod.streamToTopic[streamID]
		if !isMapped {
			topicName = core.StreamRegistry.GetStreamName(streamID)
			prod.streamToTopic[streamID] = topicName
		}

		tgo.Metric.New(kafkaMetricMessages + topicName)
		tgo.Metric.New(kafkaMetricMessagesSec + topicName)

		topicConfig := kafka.NewTopicConfig()
		topicConfig.SetI("request.required.acks", prod.topicRequiredAcks)
		topicConfig.SetI("request.timeout.ms", prod.topicTimeoutMs)
		topicConfig.SetRoundRobinPartitioner()

		topic = kafka.NewTopic(topicName, topicConfig, prod.client)

		prod.counters[topicName] = new(int64)
		prod.topic[streamID] = topic
	}

	return topic
}

func (prod *KafkaProducer) produceMessage(msg *core.Message) {
	originalMsg := *msg
	prod.ProducerBase.Format(msg)

	// Send message
	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	topic, topicMapped := prod.topic[msg.StreamID]
	if !topicMapped {
		prod.topicGuard.RUnlock()
		topic = prod.registerNewTopic(msg.StreamID)
		prod.topicGuard.RLock()
	}

	var key []byte
	if prod.keyFormat != nil {
		keyMsg := *msg
		prod.keyFormat.Format(&keyMsg)
		key = keyMsg.Data
	}

	serializedOriginal, err := originalMsg.Serialize()
	if err != nil {
		prod.Log.Error.Print(err)
	}

	kafkaMsg := &messageWrapper{
		key:   key,
		value: msg.Data,
		user:  serializedOriginal,
	}

	if err := topic.Produce(kafkaMsg); err != nil {
		prod.Log.Error.Print("Message produce failed:", err)
		prod.Drop(&originalMsg)
	} else {
		topicName := topic.GetName()
		atomic.AddInt64(prod.counters[topicName], 1)
	}
}

// OnMessageError gets called by librdkafka on message delivery failure
func (prod *KafkaProducer) OnMessageError(reason string, userdata []byte) {
	prod.Log.Error.Print("Message delivery failed:", reason)
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.Drop(&msg)
	} else {
		prod.Log.Error.Print(err)
	}
}

func (prod *KafkaProducer) poll() {
	for _, topic := range prod.topic {
		topic.Poll()
	}

	tgo.Metric.Set(kafkaMetricMessagesInflight, prod.client.GetInflightBuffers())

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	for topicName, counter := range prod.counters {
		duration := time.Since(prod.lastMetricUpdate)
		count := atomic.SwapInt64(counter, 0)
		countPerSec := int64(float64(count)/duration.Seconds() + 0.5)
		tgo.Metric.Add(kafkaMetricMessages+topicName, count)
		tgo.Metric.Set(kafkaMetricMessagesSec+topicName, countPerSec)
	}

	prod.lastMetricUpdate = time.Now()
}

func (prod *KafkaProducer) isConnected() bool {
	return prod.client != nil
}

func (prod *KafkaProducer) tryConnect() bool {
	if prod.isConnected() {
		return true
	}

	client, err := kafka.NewProducer(prod.config, prod)
	if err != nil {
		prod.Log.Error.Print(err)
		return false
	}

	prod.client = client
	return true
}

func (prod *KafkaProducer) closeConnection() {
	client := prod.client
	topics := prod.topic

	prod.client = nil
	prod.topic = make(map[core.MessageStreamID]*kafka.Topic)

	for _, topic := range topics {
		topic.Close()
	}
	client.Close()
}

func (prod *KafkaProducer) close() {
	defer prod.WorkerDone()

	for _, topic := range prod.topic {
		topic.TriggerShutdown()
	}
	prod.closeConnection()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *KafkaProducer) Produce(workers *sync.WaitGroup) {
	prod.tryConnect()
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.produceMessage, prod.pollInterval, prod.poll)
}
