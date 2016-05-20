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
	"fmt"
	kafka "github.com/trivago/gollum/contrib/native/librdkafka"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// KafkaProducer librdkafka producer plugin
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the native librdkafka (0.8.6) library so most settings relate
// to that library. This producer does not implement a fuse breaker.
// NOTICE: This producer is not included in standard builds. To enable it
// you need to trigger a custom build with native plugins enabled.
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
// KeyFormatter can define a formatter that extracts the key for a kafka message
// from the message payload. By default this is an empty string, which disables
// this feature. A good formatter for this can be format.Identifier.
//
// KeyFormatterFirst can be set to true to apply the key formatter to the
// unformatted message. By default this is set to false, so that key formatter
// uses the message after Formatter has been applied.
// KeyFormatter does never affect the payload of the message sent to kafka.
type KafkaProducer struct {
	core.ProducerBase
	servers            []string
	clientID           string
	lastMetricUpdate   time.Time
	keyFormat          core.Formatter
	client             *kafka.Client
	config             kafka.Config
	topicRequiredAcks  int
	topicTimeoutMs     int
	pollInterval       time.Duration
	topic              map[core.MessageStreamID]*topicHandle
	topicHandles       map[string]*topicHandle
	streamToTopic      map[core.MessageStreamID]string
	topicGuard         *sync.RWMutex
	keyFirst           bool
	filtersAfterFormat []core.Filter
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
	sent      int64
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
	shared.TypeRegistry.Register(KafkaProducer{})
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
func (prod *KafkaProducer) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.SetStopCallback(prod.close)
	kafka.Log = Log.Error

	if conf.HasValue("KeyFormatter") {
		keyFormatter, err := core.NewPluginWithType(conf.GetString("KeyFormatter", "format.Identifier"), conf)
		if err != nil {
			return err // ### return, plugin load error ###
		}
		prod.keyFormat = keyFormatter.(core.Formatter)
	} else {
		prod.keyFormat = nil
	}

	filters := conf.GetStringArray("FilterAfterFormat", []string{})
	for _, filterName := range filters {
		plugin, err := core.NewPluginWithType(filterName, conf)
		if err != nil {
			return err
		}
		filter, isFilter := plugin.(core.Filter)
		if !isFilter {
			return fmt.Errorf("%s is not a filter", filterName)
		}
		prod.filtersAfterFormat = append(prod.filtersAfterFormat, filter)
	}

	prod.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.lastMetricUpdate = time.Now()

	prod.topic = make(map[core.MessageStreamID]*topicHandle)
	prod.topicRequiredAcks = conf.GetInt("RequiredAcks", 1)
	prod.topicTimeoutMs = conf.GetInt("TimeoutMs", 1)
	prod.topicGuard = new(sync.RWMutex)
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")
	prod.keyFirst = conf.GetBool("KeyFormatterFirst", false)
	prod.topicHandles = make(map[string]*topicHandle)

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

	shared.Metric.New(kafkaMetricAllocations)
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
	prod.topicGuard.Lock()
	defer prod.topicGuard.Unlock()

	if topic, exists := prod.topicHandles[topicName]; exists {
		prod.topic[streamID] = topic
		return topic
	}

	topic := &topicHandle{
		handle: prod.newTopicHandle(topicName),
	}

	prod.topicHandles[topicName] = topic
	prod.topic[streamID] = topic

	shared.Metric.New(kafkaMetricMessages + topicName)
	shared.Metric.New(kafkaMetricMessagesSec + topicName)
	shared.Metric.New(kafkaMetricRoundtrip + topicName)

	return topic
}

func (prod *KafkaProducer) produceMessage(msg core.Message) {
	originalMsg := msg
	msg.Data, msg.StreamID = prod.ProducerBase.Format(msg)

	for _, filter := range prod.filtersAfterFormat {
		if !filter.Accepts(msg) {
			core.CountFilteredMessage()
			return
		}
	}

	prod.topicGuard.RLock()
	topic, topicRegistered := prod.topic[msg.StreamID]
	prod.topicGuard.RUnlock()

	if !topicRegistered {
		wildcardSet := false
		topicName, isMapped := prod.streamToTopic[msg.StreamID]
		if !isMapped {
			if topicName, wildcardSet = prod.streamToTopic[core.WildcardStreamID]; !wildcardSet {
				topicName = core.StreamRegistry.GetStreamName(msg.StreamID)
			}
		}
		topic = prod.registerNewTopic(topicName, msg.StreamID)
	}

	var key []byte
	if prod.keyFormat != nil {
		if prod.keyFirst {
			key, _ = prod.keyFormat.Format(originalMsg)
		} else {
			key, _ = prod.keyFormat.Format(msg)
		}
	}

	serializedOriginal, err := originalMsg.Serialize()
	if err != nil {
		Log.Error.Print(err)
	}

	kafkaMsg := &messageWrapper{
		key:   key,
		value: msg.Data,
		user:  serializedOriginal,
	}

	if err := topic.handle.Produce(kafkaMsg); err != nil {
		Log.Error.Print("Message produce failed:", err)
		prod.Drop(originalMsg)
	} else {
		atomic.AddInt64(&topic.sent, 1)
	}
}

func (prod *KafkaProducer) storeRTT(msg *core.Message) {
	rtt := time.Since(msg.Timestamp)
	_, streamID := prod.ProducerBase.Format(*msg)

	prod.topicGuard.RLock()
	topic := prod.topic[streamID]
	prod.topicGuard.RUnlock()

	atomic.AddInt64(&topic.rttSum, rtt.Nanoseconds()/1000)
	atomic.AddInt64(&topic.delivered, 1)
}

// OnMessageDelivered gets called by librdkafka on message delivery success
func (prod *KafkaProducer) OnMessageDelivered(userdata []byte) {
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(&msg)
	} else {
		Log.Error.Print(err)
	}
}

// OnMessageError gets called by librdkafka on message delivery failure
func (prod *KafkaProducer) OnMessageError(reason string, userdata []byte) {
	Log.Error.Print("Message delivery failed:", reason)
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(&msg)
		prod.Drop(msg)
	} else {
		Log.Error.Print(err)
	}
}

func (prod *KafkaProducer) poll() {
	prod.client.Poll(time.Second)
	shared.Metric.Set(kafkaMetricAllocations, prod.client.GetAllocCounter())

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	for _, topic := range prod.topic {
		sent := atomic.SwapInt64(&topic.sent, 0)
		duration := time.Since(prod.lastMetricUpdate)
		sentPerSec := int64(float64(sent)/duration.Seconds() + 0.5)

		rttSum := atomic.SwapInt64(&topic.rttSum, 0)
		delivered := atomic.SwapInt64(&topic.delivered, 0)
		topicName := topic.handle.GetName()

		avgRoundtripMs := int64(0)
		if delivered > 0 {
			avgRoundtripMs = rttSum / (delivered * 1000)
		}

		shared.Metric.Add(kafkaMetricMessages+topicName, sent)
		shared.Metric.Set(kafkaMetricMessagesSec+topicName, sentPerSec)
		shared.Metric.Set(kafkaMetricRoundtrip+topicName, avgRoundtripMs)
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
		Log.Error.Print(err)
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
