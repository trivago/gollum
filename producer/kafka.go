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

package producer

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	kafka "github.com/Shopify/sarama"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
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
//
// The kafka producer writes messages to a kafka cluster. This producer is
// backed by the sarama library so most settings relate to that library.
//
// Configuration example
//
//   producerKafka:
//   	type: producer.Kafka
//      ClientId: "gollum"
//      Version: "0.8.2"
//      Partitioner: "Roundrobin"
//      RequiredAcks: 1
//      TimeoutMs: 1500
//      GracePeriodMs: 10
//      SendRetries: 0
//      Compression: "None"
//      MaxOpenRequests: 5
//      MessageBufferCount: 256
//      BatchMinCount: 1
//      BatchMaxCount: 0
//      BatchSizeByte: 8192
//      BatchSizeMaxKB: 1024
//      BatchTimeoutMs: 3000
//      ServerTimeoutSec: 30
//      SendTimeoutMs: 250
//      ElectRetries: 3
//      ElectTimeoutMs: 250
//      MetadataRefreshMs: 10000
//      TlsEnabled: true
//      TlsKeyLocation: ""
//      TlsCertificateLocation: ""
//      TlsCaLocation: ""
//      TlsServerName: ""
//      TlsInsecureSkipVerify: false
//      SaslEnabled: false
//      SaslUsername: "gollum"
//      SaslPassword: ""
//      Servers:
//    	  - "localhost:9092"
//      Topic:
//        "console" : "console"
// 	KeyMetaField: ""
//
// ClientId sets the client id of this producer. By default this is "gollum".
//
// Version defines the kafka protocol version to use. Common values are 0.8.2,
// 0.9.0 or 0.10.0. Values of the form "A.B" are allowed as well as "A.B.C"
// and "A.B.C.D". Defaults to "0.8.2". If the version given is not known, the
// closest possible version is chosen.
//
// Partitioner sets the distribution algorithm to use. Valid values are:
// "Random","Roundrobin" and "Hash". By default "Roundrobin" is set.
//
// FilterAfterFormat behaves like Filter but allows filters to be executed
// after the formatter has run. By default no such filter is set.
//
// RequiredAcks defines the acknowledgment level required by the broker.
// 0 = No responses required. 1 = wait for the local commit. -1 = wait for
// all replicas to commit. >1 = wait for a specific number of commits.
// By default this is set to 1.
//
// TimeoutMs denotes the maximum time the broker will wait for acks. This
// setting becomes active when RequiredAcks is set to wait for multiple commits.
// By default this is set to 10 seconds.
//
// SendRetries defines how many times to retry sending data before marking a
// server as not reachable. By default this is set to 1.
//
// Compression sets the method of compression to use. Valid values are:
// "None","Zip" and "Snappy". By default "None" is set.
//
// MaxOpenRequests defines the number of simultaneous connections are allowed.
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
// BatchTimeoutMs sets the minimum time in milliseconds to pass after which a new
// flush will be triggered. By default this is set to 3.
//
// MessageBufferCount sets the internal channel size for the kafka client.
// By default this is set to 8192.
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
//
// GracePeriodMs defines the number of milliseconds to wait for Sarama to
// accept a single message. After this period a message is sent to the fallback.
// By default this is set to 100ms.
//
// MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
// By default this is set to 600000 (10 minutes). This corresponds to the JVM
// setting `topic.metadata.refresh.interval.ms`.
//
// TlsEnable defines whether to use TLS to communicate with brokers. Defaults
// to false.
//
// TlsKeyLocation defines the path to the client's private key (PEM) for used
// for authentication. Defaults to "".
//
// TlsCertificateLocation defines the path to the client's public key (PEM) used
// for authentication. Defaults to "".
//
// TlsCaLocation defines the path to CA certificate(s) for verifying the broker's
// key. Defaults to "".
//
// TlsServerName is used to verify the hostname on the server's certificate
// unless TlsInsecureSkipVerify is true. Defaults to "".
//
// TlsInsecureSkipVerify controls whether to verify the server's certificate
// chain and host name. Defaults to false.
//
// SaslEnable is whether to use SASL for authentication. Defaults to false.
//
// SaslUsername is the user for SASL/PLAIN authentication. Defaults to "gollum".
//
// SaslPassword is the password for SASL/PLAIN authentication. Defaults to "".
//
// Servers contains the list of all kafka servers to connect to.  By default this
// is set to contain only "localhost:9092".
//
// KeyMetaField set the message meta data key to get the kafka key from the meta data.
//
// Topic maps a stream to a specific kafka topic. You can define the
// wildcard stream (*) here, too. If defined, all streams that do not have a
// specific mapping will go to this topic (including _GOLLUM_).
// If no topic mappings are set the stream names will be used as topic.
type Kafka struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	topicGuard            *sync.RWMutex
	topic                 map[core.MessageStreamID]*topicHandle
	topicHandles          map[string]*topicHandle
	streamToTopic         map[core.MessageStreamID]string
	servers               []string      `config:"Servers" default:"localhost:9092"`
	clientID              string        `config:"ClientId" default:"gollum"`
	gracePeriod           time.Duration `config:"GracePeriodMs" default:"100" metric:"ms"`
	client                kafka.Client
	config                *kafka.Config
	producer              kafka.AsyncProducer
	missCount             int64
	nilValueAllowed       bool
	keyMetaField          string
}

type topicHandle struct {
	name          string
	rttSum        int64
	sent          int64
	delivered     int64
	lastHeartBeat time.Time
}

const (
	kafkaMetricRoundtrip = "Kafka:AvgRoundtripMs-"
)

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
		prod.config.Net.SASL.User = conf.GetString("SaslUser", "gollum")
		prod.config.Net.SASL.Password = conf.GetString("SaslPassword", "")
	}

	prod.config.Metadata.Retry.Max = int(conf.GetInt("ElectRetries", 3))
	prod.config.Metadata.Retry.Backoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	prod.config.Metadata.RefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshMs", 600000)) * time.Millisecond

	prod.config.Producer.MaxMessageBytes = int(conf.GetInt("Batch/SizeMaxKB", 1<<10)) << 10
	prod.config.Producer.RequiredAcks = kafka.RequiredAcks(conf.GetInt("RequiredAcks", int64(kafka.WaitForLocal)))
	prod.config.Producer.Timeout = time.Duration(conf.GetInt("TimoutMs", 10000)) * time.Millisecond
	prod.config.Producer.Flush.Bytes = int(conf.GetInt("Batch/SizeByte", 8192))
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

	prod.nilValueAllowed = conf.GetBool("AllowNilValue", false)
	switch strings.ToLower(conf.GetString("Partitioner", partRoundrobin)) {
	case partRandom:
		prod.config.Producer.Partitioner = kafka.NewRandomPartitioner
	case partRoundrobin:
		prod.config.Producer.Partitioner = kafka.NewRoundRobinPartitioner
	default:
		fallthrough
	case partHash:
		switch strings.ToLower(conf.GetString("PartitionHasher", "FNV-1a")) {
		case "murmur2":
			prod.config.Producer.Partitioner = NewMurmur2HashPartitioner
		default:
			fallthrough
		case "fnv-1a":
			prod.config.Producer.Partitioner = kafka.NewHashPartitioner

		}

	}

	prod.keyMetaField = conf.GetString("KeyMetaField", "")
}

func (prod *Kafka) storeRTT(msg *core.Message) {
	rtt := time.Since(msg.GetCreationTime())

	prod.topicGuard.RLock()
	topic := prod.topic[msg.GetStreamID()]
	prod.topicGuard.RUnlock()

	atomic.AddInt64(&topic.rttSum, rtt.Nanoseconds()/1000) // microseconds
	atomic.AddInt64(&topic.delivered, 1)
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
					prod.storeRTT(&msg)
				}
			}

		case err, hasMore := <-prod.producer.Errors():
			if hasMore {
				if msg, hasMsg := err.Msg.Metadata.(core.Message); hasMsg {
					prod.Logger.Warning("Kafka producer error on return: ", err)
					prod.storeRTT(&msg)
					if err == kafka.ErrMessageTooLarge {
						prod.Logger.Error("Message discarded as too large.")
						core.CountMessageDiscarded()
					} else {
						prod.TryFallback(&msg)
					}
				}
			}

		case <-timeout.C:
			keepPolling = false
		}
	}

	prod.topicGuard.RLock()
	defer prod.topicGuard.RUnlock()

	// Update metrics
	for _, topic := range prod.topic {
		rttSum := atomic.SwapInt64(&topic.rttSum, 0)
		delivered := atomic.SwapInt64(&topic.delivered, 0)
		topicName := topic.name

		avgRoundtripMs := int64(0)
		if delivered > 0 {
			avgRoundtripMs = rttSum / (delivered * 1000)
		}

		tgo.Metric.Set(kafkaMetricRoundtrip+topicName, avgRoundtripMs)
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
		name: topicName,
	}

	prod.topicHandles[topicName] = topic
	prod.topic[streamID] = topic

	tgo.Metric.New(kafkaMetricRoundtrip + topicName)

	return topic
}

func (prod *Kafka) produceMessage(msg *core.Message) {
	if !prod.nilValueAllowed && len(msg.GetPayload()) == 0 {
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

	if isConnected, err := prod.isConnected(topic.name); !isConnected {
		prod.TryFallback(msg)
		if err != nil {
			prod.Logger.Errorf("%s is not connected: %s", topic.name, err.Error())
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

	case <-timeout.C:
		// Sarama channels are full -> fallback
		prod.TryFallback(msg)
	}
}

func (prod *Kafka) getKafkaMsgKey(msg *core.Message) []byte {
	if len(prod.keyMetaField) > 0 {
		return msg.GetMetadata().GetValue(prod.keyMetaField)
	}

	return []byte{}

}

func (prod *Kafka) checkAllTopics() bool {
	topics, err := prod.client.Topics()
	if err != nil {
		prod.Logger.Error("Failed to get topics ", err.Error())
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
					prod.Logger.Debug("Broker ", broker.Addr(), " found to have an invalid connection: ", err)
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
			prod.Logger.Error("Client initialization error:", err)
			return false // ### return, connection failed ###
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		if producer, err := kafka.NewAsyncProducerFromClient(prod.client); err == nil {
			prod.producer = producer
		} else {
			prod.Logger.Error("Producer initialization error:", err)
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
