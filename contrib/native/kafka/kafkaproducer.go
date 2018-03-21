// Copyright 2015-2018 trivago N.V.
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

// +build cgo,!unit

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

// KafkaProducer producer
//
// NOTICE: This producer is not included in standard builds. To enable it
// you need to trigger a custom build with native plugins enabled.
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the native librdkafka (0.8.6) library so most settings relate
// to that library.
//
// Parameters
//
// - Servers: Defines a list of ideally all brokers in the cluster. At least one
// broker is required.
// By default this parameter is set to an empty list.
//
// - Topic: Defines a stream to topic mapping. If a stream is not mapped the
// stream name is used as topic.
// By default this parameter is set to an empty list.
//
// - ClientId: Sets the kafka client id used by this producer.
// By default this parameter is set to "gollum".
//
// - Compression: Defines the compression algorithm to use.
// Possible values are "none", "zip" and "snappy".
// By default this parameter is set to "none".
//
// - RequiredAcks: Defines the numbers of acknowledgements required until a
// message is marked as "sent".
// By default this parameter is set to 1.
//
// - ServerTimeoutSec: Defines the time in seconds after which a server is
// defined as "not reachable".
// By default this parameter is set to 1.
//
// - ServerMaxFails: Defines the number of retries after which a server is
// marked as "failing".
// By default this parameter is set to 3.
//
// - MetadataTimeoutMs: Number of milliseconds a metadata request may take until
// considered as failed.
// By default this parameter is set to 1500.
//
// - MetadataRefreshMs: Interval in milliseconds for querying metadata.
// By default this parameter is set to 300000.
//
// - TimeoutMs: Defines the number of milliseconds to wait until a request is
// marked as failed.
// By default this parameter is set to 1500.
//
// - Batch/TimeoutMs: Defines the number of milliseconds to wait until a batch
// is flushed to kafka.
// By default this parameter is set to 1000.
//
// - Batch/SizeMaxKB: Defines the maximum message size in KB.  Messages above
// this size are rejected.
// By default this parameter is set to 1024.
//
// - Batch/MinMessages: Defines the minimum number of messages required for a
// batch to be sent. This value should be significantly lower than
// BatchMaxMessages to avoid messages to be rejected.
// By default this parameter is set to 1000.
//
// - Batch/MaxMessages: Defines the maximum number of messages that are marked as
// pending at any given moment in time. If this limit is hit, additional
// messages will be rejected. This should be adjusted according to your maximum
// message throughput.
// By default this parameter is set to 100000.
//
// - SendRetries: Defines the number of times librdkafka will try to re-send a
// message if it did not succeed.
// By default this parameter is set to 0.
//
// - KeyFrom: Defines the metadata field that contains the string to be used as
// the key passed to kafka. When set to an empty string no key is used.
// By default this parameter is set to "".
//
// - SaslMechanism: Defines the SASL mechanism to use for authentication.
// Accepted values are GSSAPI, PLAIN, SCRAM-SHA-256 and SCRAM-SHA-512.
// By default this parameter is set to "".
//
// - SaslUsername: Sets the SASL username for use with the PLAIN mechanism.
// By default this parameter is set to "".
//
// - SaslPassword: Sets the SASL password for use with the PLAIN mechanism.
// By default this parameter is set to "".
//
// - SecurityProtocol: Protocol used to communicate with brokers.
// Accepted values are 	plaintext, ssl, sasl_plaintext and sasl_ssl.
// By default this parameter is set to "plaintext".
//
// - SslCipherSuites: Defines the Cipher Suites to use when connection via
// TLS/SSL. For allowed values see man page for ciphers(1).
// By default this parameter is set to "".
//
// - SslKeyLocation: Path to the client's private key (PEM) used for
// authentication.
// By default this parameter is set to "".
//
// - SslKeyPassword: Contains the private key passphrase.
// By default this parameter is set to "".
//
// - SslCertificateLocation: Path to the client's public key (PEM) used for
// authentication.
// By default this parameter is set to "".
//
// - SslCaLocation: File or directory path to the CA certificate(s) used for
// verifying the broker's key.
// By default this parameter is set to "".
//
// - SslCrlLocation: Path to the CRL used to verify the broker's certificate
// validity.
// By default this parameter is set to "".
//
// Examples:
//
//  kafkaWriter:
//    Type: native.KafkaProducer
//    Streams: logs
//    Compression: zip
//    Servers:
//    	- "kafka01:9092"
//    	- "kafka02:9092"
//    	- "kafka03:9092"
//    	- "kafka04:9092"
type KafkaProducer struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	servers               []string `config:"Servers"`
	clientID              string   `config:"ClientId" default:"gollum"`
	client                *kafka.Client
	config                kafka.Config
	topicRequiredAcks     int           `config:"RequiredAcks" default:"1"`
	topicTimeoutMs        int           `config:"TimeoutMs" default:"100" metric:"ms"`
	pollInterval          time.Duration `config:"BatchTimeoutMs" default:"1000" metric:"ms"`
	keyField              string        `config:"KeyFrom"`
	topic                 map[core.MessageStreamID]*topicHandle
	topicHandles          map[string]*topicHandle
	streamToTopic         map[core.MessageStreamID]string
	topicGuard            *sync.RWMutex
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
	prod.SetStopCallback(prod.close)

	prod.streamToTopic = conf.GetStreamMap("Topic", "default")
	prod.topic = make(map[core.MessageStreamID]*topicHandle)

	prod.topicGuard = new(sync.RWMutex)
	prod.streamToTopic = conf.GetStreamMap("Topic", "default")

	prod.topicHandles = make(map[string]*topicHandle)

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
	prod.config.SetI("message.max.bytes", int(conf.GetInt("Batch/SizeMaxKB", 1<<10))<<10)
	prod.config.SetI("metadata.request.timeout.ms", int(conf.GetInt("MetadataTimeoutMs", 1500)))
	prod.config.SetI("topic.metadata.refresh.interval.ms", int(conf.GetInt("MetadataRefreshMs", 300000)))
	prod.config.SetI("socket.max.fails", int(conf.GetInt("ServerMaxFails", 3)))
	prod.config.SetI("socket.timeout.ms", int(conf.GetInt("ServerTimeoutSec", 60)*1000))
	prod.config.SetB("socket.keepalive.enable", true)
	prod.config.SetI("message.send.max.retries", int(conf.GetInt("SendRetries", 0)))
	prod.config.SetI("queue.buffering.max.messages", int(conf.GetInt("Batch/MaxMessages", 100000)))
	prod.config.SetI("queue.buffering.max.ms", int(conf.GetInt("Batch/TimeoutMs", 1000)))
	prod.config.SetI("batch.num.messages", int(conf.GetInt("Batch/MinMessages", 1000)))
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
	if len(msg.GetPayload()) == 0 {
		streamName := core.StreamRegistry.GetStreamName(msg.GetStreamID())
		prod.Logger.Errorf("0 byte message detected on %s. Discarded", streamName)
		core.CountMessageDiscarded()
		return // ### return, invalid data ###
	}

	prod.topicGuard.RLock()
	topic, topicRegistered := prod.topic[msg.GetStreamID()]
	prod.topicGuard.RUnlock()

	if !topicRegistered {
		var wildcardSet bool
		topicName, isMapped := prod.streamToTopic[msg.GetStreamID()]
		if !isMapped {
			if topicName, wildcardSet = prod.streamToTopic[core.WildcardStreamID]; !wildcardSet {
				topicName = core.StreamRegistry.GetStreamName(msg.GetStreamID())
			}
		}
		topic = prod.registerNewTopic(topicName, msg.GetStreamID())
	}

	serializedMsg, err := msg.Serialize()
	if err != nil {
		prod.Logger.Error(err)
	}

	kafkaMsg := &messageWrapper{
		value: msg.GetPayload(),
		user:  serializedMsg,
	}

	if metadata := msg.TryGetMetadata(); metadata != nil {
		kafkaMsg.key = metadata.GetValue(prod.keyField)
	}

	if err := topic.handle.Produce(kafkaMsg); err != nil {
		prod.Logger.Error("Message produce failed:", err)
		prod.TryFallback(msg)

	} else {
		tgo.Metric.Inc(kafkaMetricMessages + topic.handle.GetName())
	}
}

func (prod *KafkaProducer) storeRTT(msg *core.Message) {
	rtt := time.Since(msg.GetCreationTime())
	prod.Modulate(msg)

	prod.topicGuard.RLock()
	topic := prod.topic[msg.GetStreamID()]
	prod.topicGuard.RUnlock()

	atomic.AddInt64(&topic.rttSum, rtt.Nanoseconds()/1000)
	atomic.AddInt64(&topic.delivered, 1)
}

// OnMessageDelivered gets called by librdkafka on message delivery success
func (prod *KafkaProducer) OnMessageDelivered(userdata []byte) {
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(msg)
	} else {
		prod.Logger.Error(err)
	}
}

// OnMessageError gets called by librdkafka on message delivery failure
func (prod *KafkaProducer) OnMessageError(reason string, userdata []byte) {
	prod.Logger.Error("Message delivery failed:", reason)
	if msg, err := core.DeserializeMessage(userdata); err == nil {
		prod.storeRTT(msg)
		prod.TryFallback(msg)
	} else {
		prod.Logger.Error(err)
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
		prod.Logger.Error(err)
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
