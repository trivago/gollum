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

package producer

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	kafka "github.com/Shopify/sarama"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/trivago/gollum/core"
)

const (
	partRandom     = "random"
	partRoundrobin = "roundrobin"
	partHash       = "hash"
	compressNone   = "none"
	compressGZIP   = "zip"
	compressSnappy = "snappy"
)

// Kafka producer
//
// This producer writes messages to a kafka cluster. This producer is backed by
// the sarama library (https://github.com/Shopify/sarama) so most settings
// directly relate to the settings of that library.
//
// Parameters
//
// - Servers: Defines a list of ideally all brokers in the cluster. At least one
// broker is required.
// By default this parameter is set to an empty list.
//
// - Version: Defines the kafka protocol version to use. Common values are 0.8.2,
// 0.9.0 or 0.10.0. Values of the form "A.B" are allowed as well as "A.B.C"
// and "A.B.C.D". If the version given is not known, the closest possible
// version is chosen. If GroupId is set to a value < "0.9", "0.9.0.1" will be used.
// By default this parameter is set to "0.8.2".
//
// - Topics: Defines a stream to topic mapping. If a stream is not mapped the
// stream name is used as topic. You can define the wildcard stream (*) here,
// too. If defined, all streams that do not have a specific mapping will go to
// this topic (including _GOLLUM_).
// By default this parameter is set to an empty list.
//
// - ClientId: Sets the kafka client id used by this producer.
// By default this parameter is set to "gollum".
//
// - Partitioner: Defines the distribution algorithm to use. Valid values are:
// Random, Roundrobin and Hash.
// By default this parameter is set to "Roundrobin".
//
// - PartitionHasher: Defines the hash algorithm to use when Partitioner is set
// to "Hash". Accepted values are "fnv1-a" and "murmur2".
//
// - KeyFrom: Defines the metadata field that contains the string to be used as
// the key passed to kafka. When set to an empty string no key is used.
// By default this parameter is set to "".
//
// - Compression: Defines the compression algorithm to use.
// Possible values are "none", "zip" and "snappy".
// By default this parameter is set to "none".
//
// - RequiredAcks: Defines the numbers of acknowledgements required until a
// message is marked as "sent". When set to -1 all replicas must acknowledge a
// message.
// By default this parameter is set to 1.
//
// - TimeoutMs: Denotes the maximum time the broker will wait for acks. This
// setting becomes active when RequiredAcks is set to wait for multiple commits.
// By default this parameter is set to 10000.
//
// - GracePeriodMs: Defines the number of milliseconds to wait for Sarama to
// accept a single message. After this period a message is sent to the fallback.
// This setting mitigates a conceptual problem in the saram API which can lead
// to long blocking times during startup.
// By default this parameter is set to 100.
//
// - MaxOpenRequests: Defines the maximum number of simultaneous connections
// opened to a single broker at a time.
// By default this parameter is set to 5.
//
// - ServerTimeoutSec: Defines the time after which a connection is set to timed
// out.
// By default this parameter is set to 30.
//
// - SendTimeoutMs: Defines the number of milliseconds to wait for a broker to
// before marking a message as timed out.
// By default this parameter is set to 250.
//
// - SendRetries: Defines how many times a message should be send again before a
// broker is marked as not reachable. Please note that this setting should never
// be 0. See https://github.com/Shopify/sarama/issues/294.
// By default this parameter is set to 1.
//
// - AllowNilValue: When enabled messages containing an empty or nil payload
// will not be rejected.
// By default this parameter is set to false.
//
// - Batch/MinCount: Sets the minimum number of messages required to send a
// request.
// By default this parameter is set to 1.
//
// - Batch/MaxCount: Defines the maximum number of messages bufferd before a
// request is sent. A value of 0 will remove this limit.
// By default this parameter is set to 0.
//
// - Batch/MinSizeByte: Defines the minimum number of bytes to buffer before
// sending a request.
// By default this parameter is set to 8192.
//
// - Batch/SizeMaxKB: Defines the maximum allowed message size in KB.
// Messages bigger than this limit will be rejected.
// By default this parameter is set to 1024.
//
// - Batch/TimeoutMs: Defines the maximum time in milliseconds after which a
// new request will be sent, ignoring of Batch/MinCount and Batch/MinSizeByte
// By default this parameter is set to 3.
//
// - ElectRetries: Defines how many times a metadata request is to be retried
// during a leader election phase.
// By default this parameter is set to 3.
//
// - ElectTimeoutMs: Defines the number of milliseconds to wait for the cluster
// to elect a new leader.
// By default this parameter is set to 250.
//
// - MetadataRefreshMs: Defines the interval in milliseconds for refetching
// cluster metadata.
// By default this parameter is set to 600000.
//
// - TlsEnable: Enables TLS communication with brokers.
// By default this parameter is set to false.
//
// - TlsKeyLocation: Path to the client's private key (PEM) used for TLS based
// authentication.
// By default this parameter is set to "".
//
// - TlsCertificateLocation: Path to the client's public key (PEM) used for TLS
// based authentication.
// By default this parameter is set to "".
//
// - TlsCaLocation: Path to the CA certificate(s) used for verifying the
// broker's key.
// By default this parameter is set to "".
//
// - TlsServerName: Used to verify the hostname on the server's certificate
// unless TlsInsecureSkipVerify is true.
// By default this parameter is set to "".
//
// - TlsInsecureSkipVerify: Enables server certificate chain and host name
// verification.
// By default this parameter is set to false.
//
// - SaslEnable: Enables SASL based authentication.
// By default this parameter is set to false.
//
// - SaslUsername: Sets the user name used for SASL/PLAIN authentication.
// By default this parameter is set to "".
//
// - SaslPassword: Sets the password used for SASL/PLAIN authentication.
// By default this parameter is set to "".
//
// MessageBufferCount sets the internal channel size for the kafka client.
// By default this is set to 8192.
//
// Examples
//
//  kafkaWriter:
//    Type: producer.Kafka
//    Streams: logs
//    Compression: zip
//    Servers:
//      - "kafka01:9092"
//      - "kafka02:9092"
//      - "kafka03:9092"
//      - "kafka04:9092"
type Kafka struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	topicGuard            *sync.RWMutex
	topic                 map[core.MessageStreamID]*topicHandle
	topicHandles          map[string]*topicHandle
	streamToTopic         map[core.MessageStreamID]string
	servers               []string      `config:"Servers"`
	clientID              string        `config:"ClientId" default:"gollum"`
	gracePeriod           time.Duration `config:"GracePeriodMs" default:"100" metric:"ms"`
	client                kafka.Client
	config                *kafka.Config
	producer              kafka.AsyncProducer
	nilValueAllowed       bool   `config:"AllowNilValue" default:"false"`
	keyField              string `config:"KeyFrom"`
	metricsRegistry       metrics.Registry
}

type topicHandle struct {
	name             string
	lastHeartBeat    time.Time
	metricsRoundtrip metrics.Timer
	metricsDelivered metrics.Counter
	metricsSent      metrics.Counter
	metricsTimeout   metrics.Counter
}

func init() {
	core.TypeRegistry.Register(Kafka{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kafka) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	kafka.Logger = prod.Logger.WithField("Scope", "Sarama")

	prod.topicGuard = new(sync.RWMutex)
	prod.streamToTopic = conf.GetStreamMap("Topics", "")
	prod.topic = make(map[core.MessageStreamID]*topicHandle)
	prod.topicHandles = make(map[string]*topicHandle)
	prod.metricsRegistry = core.NewMetricsRegistryForPlugin(prod)

	prod.config = kafka.NewConfig()
	prod.config.ClientID = prod.clientID
	prod.config.ChannelBufferSize = int(conf.GetInt("MessageBufferCount", 8192))

	switch ver := conf.GetString("Version", "0.8.2"); ver {
	case "0.8.2.0":
		prod.config.Version = kafka.V0_8_2_0
	case "0.8.2.1":
		prod.config.Version = kafka.V0_8_2_1
	case "0.8", "0.8.2", "0.8.2.2":
		prod.config.Version = kafka.V0_8_2_2
	case "0.9.0", "0.9.0.0":
		prod.config.Version = kafka.V0_9_0_0
	case "0.9", "0.9.0.1":
		prod.config.Version = kafka.V0_9_0_1
	case "0.10", "0.10.0", "0.10.0.0":
		prod.config.Version = kafka.V0_10_0_0
	default:
		prod.Logger.Warning("Unknown kafka version given: ", ver)
		parts := strings.Split(ver, ".")
		if len(parts) < 2 {
			prod.config.Version = kafka.V0_8_2_2
		} else {
			minor, _ := strconv.ParseUint(parts[1], 10, 8)
			switch {
			case minor <= 8:
				prod.config.Version = kafka.V0_8_2_2
			case minor == 9:
				prod.config.Version = kafka.V0_9_0_1
			case minor >= 10:
				prod.config.Version = kafka.V0_10_0_0
			}
		}
	}

	prod.config.Net.MaxOpenRequests = int(conf.GetInt("MaxOpenRequests", 5))
	prod.config.Net.DialTimeout = time.Duration(int(conf.GetInt("ServerTimeoutSec", 30))) * time.Second
	prod.config.Net.ReadTimeout = prod.config.Net.DialTimeout
	prod.config.Net.WriteTimeout = prod.config.Net.DialTimeout

	prod.config.Net.TLS.Enable = conf.GetBool("TlsEnable", false)
	if prod.config.Net.TLS.Enable {
		prod.config.Net.TLS.Config = &tls.Config{}

		keyFile := conf.GetString("TlsKeyLocation", "")
		certFile := conf.GetString("TlsCertificateLocation", "")
		if keyFile != "" && certFile != "" {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if conf.Errors.Push(err) {
				return
			}
			prod.config.Net.TLS.Config.Certificates = []tls.Certificate{cert}
		} else if certFile == "" {
			conf.Errors.Pushf("Cannot specify TlsKeyLocation without TlsCertificateLocation")
			return
		} else if keyFile == "" {
			conf.Errors.Pushf("Cannot specify TlsCertificateLocation without TlsKeyLocation")
			return
		}

		caFile := conf.GetString("TlsCaLocation", "")
		if caFile == "" {
			conf.Errors.Pushf("TlsEnable is set to true, but no TlsCaLocation was specified")
			return
		}
		caCert, err := ioutil.ReadFile(caFile)
		if conf.Errors.Push(err) {
			return
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		prod.config.Net.TLS.Config.RootCAs = caCertPool

		serverName := conf.GetString("TlsServerName", "")
		if serverName != "" {
			prod.config.Net.TLS.Config.ServerName = serverName
		}

		prod.config.Net.TLS.Config.InsecureSkipVerify = conf.GetBool("TlsInsecureSkipVerify", false)

	}

	prod.config.Net.SASL.Enable = conf.GetBool("SaslEnable", false)
	if prod.config.Net.SASL.Enable {
		prod.config.Net.SASL.User = conf.GetString("SaslUsername", "")
		prod.config.Net.SASL.Password = conf.GetString("SaslPassword", "")
	}

	prod.config.Metadata.Retry.Max = int(conf.GetInt("ElectRetries", 3))
	prod.config.Metadata.Retry.Backoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	prod.config.Metadata.RefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshMs", 600000)) * time.Millisecond

	prod.config.Producer.RequiredAcks = kafka.RequiredAcks(conf.GetInt("RequiredAcks", int64(kafka.WaitForLocal)))
	prod.config.Producer.Timeout = time.Duration(conf.GetInt("TimeoutMs", 10000)) * time.Millisecond
	prod.config.Producer.MaxMessageBytes = int(conf.GetInt("Batch/SizeMaxKB", 1<<10)) << 10
	prod.config.Producer.Flush.Bytes = int(conf.GetInt("Batch/MinSizeByte", 8192))
	prod.config.Producer.Flush.Messages = int(conf.GetInt("Batch/MinCount", 1))
	prod.config.Producer.Flush.Frequency = time.Duration(conf.GetInt("Batch/TimeoutMs", 3000)) * time.Millisecond
	prod.config.Producer.Flush.MaxMessages = int(conf.GetInt("Batch/MaxCount", 0))
	prod.config.Producer.Retry.Max = int(conf.GetInt("SendRetries", 1))
	prod.config.Producer.Retry.Backoff = time.Duration(conf.GetInt("SendTimeoutMs", 100)) * time.Millisecond

	prod.config.Producer.Return.Successes = true
	prod.config.Producer.Return.Errors = true
	prod.config.Producer.Return.Successes = true

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
	case partHash:
		fallthrough
	default:
		switch strings.ToLower(conf.GetString("PartitionHasher", "FNV-1a")) {
		case "murmur2":
			prod.config.Producer.Partitioner = NewMurmur2HashPartitioner
		case "fnv-1a":
			fallthrough
		default:
			prod.config.Producer.Partitioner = kafka.NewHashPartitioner

		}
	}
}

func (prod *Kafka) onMsgReturned(msg *core.Message) {
	prod.topicGuard.RLock()
	topic := prod.topic[msg.GetStreamID()]
	prod.topicGuard.RUnlock()

	topic.metricsRoundtrip.UpdateSince(msg.GetCreationTime())
	topic.metricsDelivered.Inc(1)
}

func (prod *Kafka) pollResults() {
	// Check for results
	keepPolling := true
	timeout := time.NewTimer(prod.config.Producer.Flush.Frequency / 2)
	for keepPolling && prod.producer != nil {
		select {
		case result, hasMore := <-prod.producer.Successes():
			if hasMore {
				if msg, hasMsg := result.Metadata.(core.Message); hasMsg {
					prod.onMsgReturned(&msg)
				}
			}

		case err, hasMore := <-prod.producer.Errors():
			if hasMore {
				if msg, hasMsg := err.Msg.Metadata.(core.Message); hasMsg {
					prod.Logger.WithError(err).Warning("Kafka producer error on return: ")
					prod.onMsgReturned(&msg)
					if err == kafka.ErrMessageTooLarge {
						prod.Logger.Error("Message discarded as too large.")
						core.MetricMessagesDiscarded.Inc(1)
					} else {
						prod.TryFallback(&msg)
					}
				}
			}

		case <-timeout.C:
			keepPolling = false
		}
	}
}

func (prod *Kafka) registerNewTopic(topicName string, streamID core.MessageStreamID) *topicHandle {
	prod.topicGuard.Lock()
	defer prod.topicGuard.Unlock()

	if topic, exists := prod.topicHandles[topicName]; exists {
		prod.topic[streamID] = topic
		return topic
	}

	topic := &topicHandle{
		name:             topicName,
		metricsSent:      metrics.NewCounter(),
		metricsDelivered: metrics.NewCounter(),
		metricsTimeout:   metrics.NewCounter(),
		metricsRoundtrip: metrics.NewTimer(),
	}

	prod.topicHandles[topicName] = topic
	prod.topic[streamID] = topic

	prod.metricsRegistry.Register(topicName+".sent", topic.metricsSent)
	prod.metricsRegistry.Register(topicName+".delivered", topic.metricsDelivered)
	prod.metricsRegistry.Register(topicName+".rtt", topic.metricsRoundtrip)
	prod.metricsRegistry.Register(topicName+".timeout", topic.metricsTimeout)

	return topic
}

func (prod *Kafka) produceMessage(msg *core.Message) {
	if !prod.nilValueAllowed && len(msg.GetPayload()) == 0 {
		streamName := core.StreamRegistry.GetStreamName(msg.GetStreamID())
		prod.Logger.Errorf("0 byte message detected on %s. Discarded", streamName)
		core.MetricMessagesDiscarded.Inc(1)
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

	if isConnected, err := prod.isConnected(topic.name); !isConnected {
		prod.TryFallback(msg)
		if err != nil {
			prod.Logger.WithError(err).Errorf("Topic %s is not connected", topic.name)
		}
		// TBD: health check? (ex-fuse breaker)
		return // ### return, not connected ###
	}

	kafkaMsg := &kafka.ProducerMessage{
		Topic:    topic.name,
		Value:    kafka.ByteEncoder(msg.GetPayload()),
		Metadata: &msg,
	}

	kafkaKey := prod.getKafkaMsgKey(msg)
	if len(kafkaKey) > 0 {
		kafkaMsg.Key = kafka.ByteEncoder(kafkaKey)
	}

	// Sarama can block on single messages if all buffers are full.
	// So we stop trying after a few milliseconds
	timeout := time.NewTimer(prod.gracePeriod)
	select {
	case prod.producer.Input() <- kafkaMsg:
		timeout.Stop()
		topic.metricsSent.Inc(1)

	case <-timeout.C:
		// Sarama channels are full -> fallback
		prod.TryFallback(msg)
		topic.metricsTimeout.Inc(1)
	}
}

func (prod *Kafka) getKafkaMsgKey(msg *core.Message) []byte {
	if len(prod.keyField) > 0 {
		if metadata := msg.TryGetMetadata(); metadata != nil {
			key, _ := metadata.Value(prod.keyField)
			return core.ConvertToBytes(key)
		}
	}

	return []byte{}

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

	prod.topicGuard.RLock()
	handle := prod.topicHandles[topic]
	prod.topicGuard.RUnlock()

	doHeartBeat := time.Since(handle.lastHeartBeat) > prod.config.Net.DialTimeout
	if doHeartBeat {
		defer func() { handle.lastHeartBeat = time.Now() }()
	}

	for _, p := range partitions {
		broker, err := prod.client.Leader(topic, p)
		if err != nil {
			return false, err // ### return, error ###
		}

		// Do a heartbeat to check if connection is functional
		if doHeartBeat {
			if connected, _ := broker.Connected(); connected {
				req := &kafka.MetadataRequest{Topics: []string{topic}}
				if _, err := broker.GetMetadata(req); err != nil {
					prod.Logger.WithError(err).Debugf("Server %s found to have an invalid connection", broker.Addr())
					broker.Close()
				}
			}
		}

		// Reconnect if necessary
		if connected, _ := broker.Connected(); !connected {
			if errOpen := broker.Open(prod.config); errOpen != nil {
				return false, errOpen
			}
		}
	}

	return true, nil
}

func (prod *Kafka) tryOpenConnection() bool {
	// Reconnect the client first
	if prod.client == nil {
		if client, err := kafka.NewClient(prod.servers, prod.config); err == nil {
			prod.client = client
		} else {
			prod.Logger.WithError(err).Error("Client initialization error")
			return false // ### return, connection failed ###
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		if producer, err := kafka.NewAsyncProducerFromClient(prod.client); err == nil {
			prod.producer = producer
		} else {
			prod.Logger.WithError(err).Error("Producer initialization error")
			return false // ### return, connection failed ###
		}
	}

	return true
}

func (prod *Kafka) closeConnection() {
	if prod.producer != nil {
		prod.producer.Close()
	}
	if prod.client != nil {
		prod.client.Close()
	}
}

func (prod *Kafka) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()
	prod.closeConnection()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *Kafka) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.tryOpenConnection()
	prod.TickerMessageControlLoop(prod.produceMessage, prod.config.Producer.Flush.Frequency, prod.pollResults)
}
