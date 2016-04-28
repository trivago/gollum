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
// library. This producer does not implement a fuse breaker.
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
//    BatchMinMessages: 1000
//    BatchTimeoutMs: 1000
//    ServerTimeoutSec: 60
//    ServerMaxFails: 3
//    MetadataTimeoutMs: 1500
//    MetadataRefreshMs: 300000
//    KeyFormatter: ""
//    Servers:
//    	- "localhost:9092"
//    Topic:
//      "console" : "console"
//
// SendRetries is mapped to message.send.max.retries.
// This defines the number of times librdkafka will try to re-send a message
// if it did not succeed. Set to 0 by default (don't retry).
//
// Compression is mapped to compression.codec. Please note that "zip" has to be
// used instead of "gzip". Possible values are "none", "zip" and "snappy".
// By default this is set to "none".
//
// TimeoutMs is mapped to request.timeout.ms.
// This defines the number of milliseconds to wait until a request is marked
// as failed. By default this is set to 1.5sec.
//
// BatchSizeMaxKB is mapped to message.max.bytes (x1024).
// This defines the maximum message size in KB. By default this is set to 1 MB.
// Messages above this size are rejected.
//
// BatchMaxMessages is mapped to queue.buffering.max.messages.
// This defines the maximum number of messages that can be pending at any given
// moment in time. If this limit is hit additional messages will be rejected.
// This value is set to 100.000 by default and should be adjusted according to
// your average message throughput.
//
// BatchMinMessages is mapped to batch.num.messages.
// This defines the minimum number of messages required for a batch to be sent.
// This is set to 1000 by default and should be significantly lower than
// BatchMaxMessages to avoid messages to be rejected.
//
// BatchTimeoutMs is mapped to queue.buffering.max.ms.
// This defines the number of milliseconds to wait until a batch is flushed to
// kafka. Set to 1sec by default.
//
// ServerTimeoutSec is mapped to socket.timeout.ms.
// Defines the time in seconds after a server is defined as "not reachable".
// Set to 1 minute by default.
//
// ServerMaxFails is mapped to socket.max.fails.
// Number of retries after a server is marked as "failing".
//
// MetadataTimeoutMs is mapped to metadata.request.timeout.ms.
// Number of milliseconds a metadata request may take until considered as failed.
// Set to 1.5 seconds by default.
//
// MetadataRefreshMs is mapped to topic.metadata.refresh.interval.ms.
// Interval in milliseconds for querying metadata. Set to 5 minutes by default.
//
// Servers defines the list of brokers to produce messages to.
//
// Topic defines a stream to topic mapping.
// If a stream is not mapped a topic named like the stream is assumed.
//
// KeyFormatter defines the formatter used to extract keys from a message.
// Set to "" by default (disable).
type KafkaProducer struct {
	core.BufferedProducer
	servers           []string
	clientID          string
	keyFormat         core.Formatter
	client            *kafka.Client
	config            kafka.Config
	topicRequiredAcks int
	topicTimeoutMs    int
	pollInterval      time.Duration
	topic             map[core.MessageStreamID]*topicHandle
	streamToTopic     map[core.MessageStreamID]string
	topicGuard        *sync.RWMutex
}

type messageWrapper struct {
	key   []byte
	value []byte
	user  []byte
}

type topicHandle struct {
	handle    *kafka.Topic
	rttSum    int64
	delivered int64
}

const (
	kafkaMetricMessages    = "Kafka:Messages-"
	kafkaMetricMessagesSec = "Kafka:MessagesSec-"
	kafkaMetricRoundtrip   = "Kafka:AvgRoundtripMs-"
	kafkaMetricAllocations = "Kafka:Allocations"
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
	prod.BufferedProducer.Configure(conf)

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
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")
	prod.topic = make(map[core.MessageStreamID]*topicHandle)
	prod.topicRequiredAcks = conf.GetInt("RequiredAcks", 1)
	prod.topicTimeoutMs = conf.GetInt("TimeoutMs", 1)
	prod.topicGuard = new(sync.RWMutex)
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")

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
	prod.config.SetI("metadata.request.timeout.ms", int(conf.GetInt("MetadataTimeoutMs", 1500)))
	prod.config.SetI("topic.metadata.refresh.interval.ms", int(conf.GetInt("MetadataRefreshMs", 300000)))
	prod.config.SetI("socket.max.fails", int(conf.GetInt("ServerMaxFails", 3)))
	prod.config.SetI("socket.timeout.ms", int(conf.GetInt("ServerTimeoutSec", 60)*1000))
	prod.config.SetB("socket.keepalive.enable", true)
	prod.config.SetI("message.send.max.retries", conf.GetInt("SendRetries", 0))
	prod.config.SetI("queue.buffering.max.messages", conf.GetInt("BatchMaxMessages", 100000))
	prod.config.SetI("queue.buffering.max.ms", batchIntervalMs)
	prod.config.SetI("batch.num.messages", conf.GetInt("BatchMinMessages", 1000))
	//prod.config.SetI("protocol.version", verNumber)

	switch strings.ToLower(conf.GetString("Compression", compressNone)) {
	default:
		prod.config.Set("compression.codec", "none")
	case compressGZIP:
		prod.config.Set("compression.codec", "gzip")
	case compressSnappy:
		prod.config.Set("compression.codec", "snappy")
	}

	tgo.Metric.New(kafkaMetricAllocations)
	return nil
}

func (prod *KafkaProducer) newTopicHandle(topicName string) *kafka.Topic {
	topicConfig := kafka.NewTopicConfig()
	topicConfig.SetI("request.required.acks", prod.topicRequiredAcks)
	topicConfig.SetI("request.timeout.ms", prod.topicTimeoutMs)
	topicConfig.SetRoundRobinPartitioner()

	return kafka.NewTopic(topicName, topicConfig, prod.client)
}

func (prod *KafkaProducer) registerNewTopic(topicName string, streamID core.MessageStreamID) *topicHandle {
	topic := &topicHandle{
		handle: prod.newTopicHandle(topicName),
	}

	tgo.Metric.New(kafkaMetricMessages + topicName)
	tgo.Metric.New(kafkaMetricMessagesSec + topicName)
	tgo.Metric.New(kafkaMetricRoundtrip + topicName)

	prod.topicGuard.Lock()
	prod.topic[streamID] = topic
	prod.topicGuard.Unlock()

	return topic
}

func (prod *KafkaProducer) produceMessage(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	// Send message
	prod.topicGuard.RLock()
	topic, topicRegistered := prod.topic[msg.StreamID]

	if !topicRegistered {
		prod.topicGuard.RUnlock()
		topicName, isMapped := prod.streamToTopic[msg.StreamID]
		if !isMapped {
			topicName = core.StreamRegistry.GetStreamName(msg.StreamID)
		}
		topic = prod.registerNewTopic(topicName, msg.StreamID)
	} else {
		prod.topicGuard.RUnlock()
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

	if err := topic.handle.Produce(kafkaMsg); err != nil {
		prod.Log.Error.Print("Message produce failed:", err)
		prod.Drop(&originalMsg)
	} else {
		tgo.Metric.Inc(kafkaMetricMessages + topic.handle.GetName())
	}
}

func (prod *KafkaProducer) storeRTT(msg *core.Message) {
	rtt := time.Since(msg.Timestamp)

	prod.topicGuard.RLock()
	topic := prod.topic[msg.StreamID]
	prod.topicGuard.RUnlock()

	atomic.AddInt64(&topic.rttSum, rtt.Nanoseconds()/1000)
	atomic.AddInt64(&topic.delivered, 1)
}

// OnMessageDelivered gets called by librdkafka on message delivery success
func (prod *KafkaProducer) OnMessageDelivered(userdata []byte) {
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(&msg)
	} else {
		prod.Log.Error.Print(err)
	}
}

// OnMessageError gets called by librdkafka on message delivery failure
func (prod *KafkaProducer) OnMessageError(reason string, userdata []byte) {
	prod.Log.Error.Print("Message delivery failed:", reason)
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(&msg)
		prod.Drop(&msg)
	} else {
		prod.Log.Error.Print(err)
	}
}

func (prod *KafkaProducer) poll() {
	prod.client.Poll(time.Second)
	tgo.Metric.Set(kafkaMetricAllocations, prod.client.GetAllocCounter())

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	for _, topic := range prod.topic {
		rttSum := atomic.SwapInt64(&topic.rttSum, 0)
		delivered := atomic.SwapInt64(&topic.delivered, 0)
		topicName := topic.handle.GetName()

		avgRoundtripMs := int64(0)
		if delivered > 0 {
			avgRoundtripMs = rttSum / (delivered * 1000)
		}

		tgo.Metric.Set(kafkaMetricRoundtrip+topicName, avgRoundtripMs)
	}
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

	prod.topicGuard.Lock()
	defer prod.topicGuard.Unlock()

	prod.client = client
	for _, topic := range prod.topic {
		topic.handle = prod.newTopicHandle(topic.handle.GetName())
	}
	return true
}

func (prod *KafkaProducer) close() {
	defer prod.WorkerDone()

	prod.CloseMessageChannel(prod.produceMessage)

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	for _, topic := range prod.topic {
		topic.handle.Close()
	}

	prod.client.Close()
	prod.client = nil
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *KafkaProducer) Produce(workers *sync.WaitGroup) {
	prod.tryConnect()
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.produceMessage, prod.pollInterval, prod.poll)
}
