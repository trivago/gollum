// Copyright 2015-2017 trivago GmbH
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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	kafka "github.com/trivago/gollum/contrib/native/kafka/librdkafka"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
)

// KafkaProducer librdkafka producer plugin
//
// NOTICE: This producer is not included in standard builds. To enable it
// you need to trigger a custom build with native plugins enabled.
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the native librdkafka (0.8.6) library so most settings relate
// to that library.
//
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
//    SecurityProtocol: "plaintext"
//    SslCipherSuites: ""
//    SslKeyLocation: ""
//    SslKeyPassword: ""
//    SslCertificateLocation: ""
//    SslCaLocation: ""
//    SslCrlLocation: ""
//    SaslMechanism: ""
//    SaslUsername: ""
//    SaslPassword: ""
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
// SecurityProtocol is mapped to security.protocol.
// Protocol used to communicate with brokers. Set to plaintext by default.
//
// SslCipherSuites is mapped to ssl.cipher.suites.
// Cipher Suites to use when connection via TLS/SSL. Not set by default.
//
// SslKeyLocation is mapped to ssl.key.location.
// Path to client's private key (PEM) for used for authentication. Not set by default.
//
// SslKeyPassword is mapped to ssl.key.password.
// Private key passphrase. Not set by default.
//
// SslCertificateLocation is mapped to ssl.certificate.location.
// Path to client's public key (PEM) used for authentication. Not set by default.
//
// SslCaLocation is mapped to ssl.ca.location.
// File or directory path to CA certificate(s) for verifying the broker's key. Not set by default.
//
// SslCrlLocation is mapped to ssl.crl.location.
// Path to CRL for verifying broker's certificate validity. Not set by default.
//
// SaslMechanism is mapped to sasl.mechanisms.
// SASL mechanism to use for authentication. Not set by default.
//
// SaslUsername is mapped to sasl.username.
// SASL username for use with the PLAIN mechanism. Not set by default.
//
// SaslPassword is mapped to sasl.password.
// SASL password for use with the PLAIN mechanism. Not set by default.
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
//
// FilterAfterFormat behaves like Filter but allows filters to be executed
// after the formatter has run. By default no such filter is set.
type KafkaProducer struct {
	core.BufferedProducer
	servers           []string
	clientID          string
	keyModulators     core.ModulatorArray
	client            *kafka.Client
	config            kafka.Config
	topicRequiredAcks int
	topicTimeoutMs    int
	pollInterval      time.Duration
	topic             map[core.MessageStreamID]*topicHandle
	topicHandles      map[string]*topicHandle
	streamToTopic     map[core.MessageStreamID]string
	topicGuard        *sync.RWMutex
	keyFirst          bool
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

const (
	protocolPlaintext     = "plaintext"
	protocolSsl           = "ssl"
	protocolSaslPlaintext = "sasl_plaintext"
	protocolSaslSsl       = "sasl_ssl"
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
	prod.keyModulators = conf.GetModulatorArray("KeyModulators", prod.Log, core.ModulatorArray{})

	prod.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")
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
	prod.config.Set("sasl.mechanisms", conf.GetString("SaslMechanism", ""))
	prod.config.Set("sasl.password", conf.GetString("SaslPassword", ""))
	prod.config.Set("sasl.username", conf.GetString("SaslUsername", ""))
	prod.config.Set("ssl.ca.location", conf.GetString("SslCaLocation", ""))
	prod.config.Set("ssl.certificate.location", conf.GetString("SslCertificateLocation", ""))
	prod.config.Set("ssl.cipher.suites", conf.GetString("SslCipherSuites", ""))
	prod.config.Set("ssl.crl.location", conf.GetString("SslCrlLocation", ""))
	prod.config.Set("ssl.key.location", conf.GetString("SslKeyLocation", ""))
	prod.config.Set("ssl.key.password", conf.GetString("SslKeyPassword", ""))
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

	securityProtocol := strings.ToLower(conf.GetString("SecurityProtocol", protocolPlaintext))
	switch securityProtocol {
	default:
		return fmt.Errorf("%s is not a valid security protocol", securityProtocol)
	case protocolPlaintext:
		prod.config.Set("security.protocol", protocolPlaintext)
	case protocolSsl:
		prod.config.Set("security.protocol", protocolSsl)
	case protocolSaslPlaintext:
		prod.config.Set("security.protocol", protocolSaslPlaintext)
	case protocolSaslSsl:
		prod.config.Set("security.protocol", protocolSaslSsl)
	}

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

	tgo.Metric.New(kafkaMetricMessages + topicName)
	tgo.Metric.New(kafkaMetricMessagesSec + topicName)
	tgo.Metric.New(kafkaMetricRoundtrip + topicName)

	return topic
}

func (prod *KafkaProducer) produceMessage(msg *core.Message) {
	originalMsg := msg.Clone()

	if msg.Len() == 0 {
		streamName := core.StreamRegistry.GetStreamName(msg.StreamID())
		prod.Log.Error.Printf("0 byte message detected on %s. Discarded", streamName)
		core.CountDiscardedMessage()
		return // ### return, invalid data ###
	}

	prod.topicGuard.RLock()
	topic, topicRegistered := prod.topic[msg.StreamID()]
	prod.topicGuard.RUnlock()

	if !topicRegistered {
		wildcardSet := false
		topicName, isMapped := prod.streamToTopic[msg.StreamID()]
		if !isMapped {
			if topicName, wildcardSet = prod.streamToTopic[core.WildcardStreamID]; !wildcardSet {
				topicName = core.StreamRegistry.GetStreamName(msg.StreamID())
			}
		}
		topic = prod.registerNewTopic(topicName, msg.StreamID())
	}

	serializedOriginal, err := originalMsg.Serialize()
	if err != nil {
		prod.Log.Error.Print(err)
	}

	kafkaMsg := &messageWrapper{
		key:   []byte{},
		value: msg.Data(),
		user:  serializedOriginal,
	}

	if prod.keyFirst {
		keyMsg := originalMsg.Clone()
		prod.keyModulators.Modulate(keyMsg)
		kafkaMsg.key = keyMsg.Data()
	} else {
		keyMsg := msg.Clone()
		prod.keyModulators.Modulate(keyMsg)
		kafkaMsg.key = keyMsg.Data()
	}

	if err := topic.handle.Produce(kafkaMsg); err != nil {
		prod.Log.Error.Print("Message produce failed:", err)
		prod.Drop(originalMsg)
	} else {
		tgo.Metric.Inc(kafkaMetricMessages + topic.handle.GetName())
	}
}

func (prod *KafkaProducer) storeRTT(msg *core.Message) {
	rtt := time.Since(msg.Created())
	prod.Modulate(msg)

	prod.topicGuard.RLock()
	topic := prod.topic[msg.StreamID()]
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
