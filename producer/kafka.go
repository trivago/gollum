// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	kafka "gopkg.in/Shopify/sarama.v1"
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
//   - "producer.Kafka":
//     Enable: true
//     ClientId: "weblog"
//     Partitioner: "Roundrobin"
//     RequiredAcks: 0
//     TimeoutMs: 0
//     Compression: "Snappy"
//     BatchMinCount: 10
//     BatchSizeByte: 16384
//     BatchTimeoutSec: 5
//     BufferSizeMaxKB: 524288
//     BatchMaxCount: 0
//     ElectTimeoutMs: 1000
//     MetadataRefreshSec: 30
//     Servers:
//     	- "192.168.222.30:9092"
//     Stream:
//       - "console"
//       - "_GOLLUM_"
//     Topic:
//       "console" : "default"
//       "_GOLLUM_"  : "default"
//
// ClientId sets the client id of this producer. By default this is "gollum".
//
// MessageBufferCount sets the internal channel size for the kafka client.
// By default this is set to 256.
//
// MaxOpenRequests defines the number of simultanious connections are allowed.
// By default this is set to 5.
//
// ServerTimeoutSec defines the time after which a connection is set to timed
// out. By default this is set to 30 seconds.
//
// BufferSizeMaxKB defines the maximum allowed message size. By default this is
// set to 1 MB.
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
// Partitioner sets the distribution algorithm to use. Valid values (case
// sensitive) are: "Random","Roundrobin","Hash". By default "Hash" is set.
//
// BatchSizeByte sets the mimimum number of bytes to collect before a new flush
// is triggered. By default this is set to 8192.
//
// BatchMinCount sets the minimum number of messages required to trigger a
// flush. By default this is set to 1.
//
// BatchTimeoutSec sets the minimum time in seconds to pass after wich a new
// flush will be triggered. By default this is set to 3.
//
// BatchMaxCount defines the maximum number of messages processed per
// request. By default this is set to 0 for "unlimited".
//
// SendRetries defines how many times to retry sending data before marking a
// server as not reachable. By default this is set to 3.
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
// MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
// By default this is set to 10000. This corresponds to the JVM setting
// `topic.metadata.refresh.interval.ms`.
//
// Servers contains the list of all kafka servers to connect to. This setting
// is mandatory and has no defaults.
//
// Topic maps a stream to a specific kafka topic. You can define the
// wildcard stream (*) here, too. All streams that do not have a specific
// mapping will go to this stream (including _GOLLUM_).
// If no topic mappings are set all messages will be send to "default".
type Kafka struct {
	core.ProducerBase
	servers  []string
	topic    map[core.MessageStreamID]string
	clientID string
	client   kafka.Client
	config   *kafka.Config
	producer kafka.AsyncProducer
}

func init() {
	shared.RuntimeType.Register(Kafka{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kafka) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("Servers") {
		return core.NewProducerError("No servers configured for producer.Kafka")
	}

	prod.servers = conf.GetStringArray("Servers", []string{})
	prod.topic = conf.GetStreamMap("Topic", "default")
	prod.clientID = conf.GetString("ClientId", "gollum")

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

	prod.config.Producer.MaxMessageBytes = conf.GetInt("BufferSizeMaxKB", 1<<10) << 10
	prod.config.Producer.RequiredAcks = kafka.RequiredAcks(conf.GetInt("RequiredAcks", int(kafka.WaitForLocal)))
	prod.config.Producer.Timeout = time.Duration(conf.GetInt("TimoutMs", 1500)) * time.Millisecond

	prod.config.Producer.Return.Errors = true
	prod.config.Producer.Return.Successes = false

	switch conf.GetString("Compression", compressNone) {
	default:
		fallthrough
	case compressNone:
		prod.config.Producer.Compression = kafka.CompressionNone
	case compressGZIP:
		prod.config.Producer.Compression = kafka.CompressionGZIP
	case compressSnappy:
		prod.config.Producer.Compression = kafka.CompressionSnappy
	}

	switch conf.GetString("Partitioner", partRandom) {
	case partRandom:
		prod.config.Producer.Partitioner = kafka.NewRandomPartitioner
	case partRoundrobin:
		prod.config.Producer.Partitioner = kafka.NewRoundRobinPartitioner
	default:
		fallthrough
	case partHash:
		prod.config.Producer.Partitioner = kafka.NewHashPartitioner
	}

	prod.config.Producer.Flush.Bytes = conf.GetInt("BatchSizeByte", 8192)
	prod.config.Producer.Flush.Messages = conf.GetInt("BatchMinCount", 1)
	prod.config.Producer.Flush.Frequency = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second
	prod.config.Producer.Flush.MaxMessages = conf.GetInt("BatchMaxCount", 0)
	prod.config.Producer.Retry.Max = conf.GetInt("SendRetries", 3)
	prod.config.Producer.Retry.Backoff = time.Duration(conf.GetInt("SendTimeoutMs", 100)) * time.Millisecond
	return nil
}

func (prod *Kafka) send(msg core.Message) {
	// If we have not yet connected or the connection dropped: connect.
	if prod.client == nil || prod.client.Closed() {
		if prod.producer != nil {
			prod.producer.Close()
			prod.producer = nil
		}

		var err error
		prod.client, err = kafka.NewClient(prod.servers, prod.config)
		if err != nil {
			Log.Error.Print("Kafka client error:", err)
			return // ### return, connection failed ###
		}
	}

	// Make sure we have a producer up and running
	if prod.producer == nil {
		var err error
		prod.producer, err = kafka.NewAsyncProducerFromClient(prod.client)
		if err != nil {
			Log.Error.Print("Kafka producer error:", err)
			return // ### return, connection failed ###
		}
	}

	if prod.client != nil && prod.producer != nil {
		// Send message
		topic, topicMapped := prod.topic[msg.CurrentStream]
		if !topicMapped {
			topic = prod.topic[core.WildcardStreamID]
		}

		prod.Formatter().PrepareMessage(msg)
		buffer := make([]byte, prod.Formatter().Len())
		prod.Formatter().Read(buffer)

		prod.producer.Input() <- &kafka.ProducerMessage{
			Topic: topic,
			Key:   nil,
			Value: kafka.ByteEncoder(buffer),
		}

		// Check for errors
		select {
		case err := <-prod.producer.Errors():
			Log.Error.Print("Kafka producer error:", err)
		default:
		}
	}
}

// Produce writes to a buffer that is sent to a given socket.
func (prod Kafka) Produce(workers *sync.WaitGroup) {
	defer func() {
		if prod.producer != nil {
			prod.producer.Close()
		}
		if prod.client != nil && !prod.client.Closed() {
			prod.client.Close()
		}
		prod.WorkerDone()
	}()

	prod.AddMainWorker(workers)
	prod.DefaultControlLoop(prod.send, nil)
}
