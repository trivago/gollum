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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	//"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type KafkaProducer struct {
	core.ProducerBase
	servers          []string
	clientID         string
	batch            core.MessageBatch
	counters         map[string]*int64
	lastMetricUpdate time.Time
	batchTimeout     time.Duration
	keyFormat        core.Formatter
	client           *kafka.Client
	config           kafka.Config
	topicConfig      kafka.TopicConfig
	streamToTopic    map[core.MessageStreamID]string
	topic            map[core.MessageStreamID]*kafka.Topic
}

const (
	kafkaMetricMessages    = "Kafka:Messages-"
	kafkaMetricMessagesSec = "Kafka:MessagesSec-"
)

const (
	compressNone   = "none"
	compressGZIP   = "zip"
	compressSnappy = "snappy"
)

func init() {
	shared.TypeRegistry.Register(KafkaProducer{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *KafkaProducer) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	if conf.HasValue("KeyFormatter") {
		keyFormatter, err := core.NewPluginWithType(conf.GetString("KeyFormatter", "format.Identifier"), conf)
		if err != nil {
			return err // ### return, plugin load error ###
		}
		prod.keyFormat = keyFormatter.(core.Formatter)
	} else {
		prod.keyFormat = nil
	}

	prod.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.lastMetricUpdate = time.Now()
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second

	prod.batch = core.NewMessageBatch(conf.GetInt("Channel", 8192))
	prod.counters = make(map[string]*int64)
	prod.streamToTopic = conf.GetStreamMap("Topics", "default")
	prod.topic = make(map[core.MessageStreamID]*kafka.Topic)

	/*compression := "none"
	switch strings.ToLower(conf.GetString("Compression", compressNone)) {
	case compressGZIP:
		compression = "gzip"
	case compressSnappy:
		compression = "snappy"
	}*/

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

	if err := prod.config.Set("client.id", conf.GetString("ClientId", "gollum")); err != nil {
		return err
	}
	if err := prod.config.Set("metadata.broker.list", strings.Join(prod.servers, ",")); err != nil {
		return err
	}
	if err := prod.config.SetI("message.max.bytes", conf.GetInt("BatchSizeMaxKB", 1<<10)<<10); err != nil {
		return err
	}
	if err := prod.config.SetI("metadata.request.timeout.ms", int(conf.GetInt("MetadataTimeoutSec", 60)*1000)); err != nil {
		return err
	}
	if err := prod.config.SetI("topic.metadata.refresh.interval.ms", int(conf.GetInt("MetadataRefreshMs", 300000))); err != nil {
		return err
	}
	if err := prod.config.SetI("socket.max.fails", int(conf.GetInt("ServerMaxFails", 3))); err != nil {
		return err
	}
	if err := prod.config.SetI("socket.timeout.ms", int(conf.GetInt("ServerTimeoutSec", 60)*1000)); err != nil {
		return err
	}
	if err := prod.config.SetB("socket.keepalive.enable", true); err != nil {
		return err
	}
	if err := prod.config.SetI("message.send.max.retries", conf.GetInt("SendRetries", 0)); err != nil {
		return err
	}
	if err := prod.config.SetI("queue.buffering.max.messages", conf.GetInt("BatchMaxMessages", 100000)); err != nil {
		return err
	}
	if err := prod.config.SetI("queue.buffering.max.ms", conf.GetInt("BatchTimeoutMs", 10)); err != nil {
		return err
	}
	if err := prod.config.SetI("batch.num.messages", prod.batch.Len()); err != nil {
		return err
	}
	/*if err := prod.config.Set("partition.assignment.strategy", "roundrobin"); err != nil {
		return err
	}
	if err := prod.config.SetI("protocol.version", verNumber); err != nil {
		return err
	}*/

	// Topic
	prod.topicConfig = kafka.NewTopicConfig()
	if err := prod.topicConfig.SetI("request.required.acks", conf.GetInt("RequiredAcks", 1)); err != nil {
		return err
	}
	if err := prod.topicConfig.SetI("request.timeout.ms", int(conf.GetInt("TimeoutMs", 1))); err != nil {
		return err
	}
	/*if err := prod.topicConfig.Set("compression.codec", compression); err != nil {
		return err
	}*/

	return nil
}

func (prod *KafkaProducer) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *KafkaProducer) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}

	// Update metrics
	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for topic, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)
		shared.Metric.Add(kafkaMetricMessages+topic, count)
		shared.Metric.SetF(kafkaMetricMessagesSec+topic, float64(count)/duration.Seconds())
	}
}

func (prod *KafkaProducer) sendBatch() {
	if prod.tryConnect() {
		prod.batch.Flush(prod.transformMessages)
	} else if !prod.IsStopping() {
		prod.batch.Flush(prod.dropMessages)
	}
}

func (prod *KafkaProducer) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *KafkaProducer) transformMessages(messages []core.Message) {
	batch := make(map[*kafka.Topic][]kafka.Message)

	for _, msg := range messages {
		originalMsg := msg
		msg.Data, msg.StreamID = prod.ProducerBase.Format(msg)

		// Send message
		topic, topicMapped := prod.topic[msg.StreamID]
		if !topicMapped {
			topicName, isMapped := prod.streamToTopic[msg.StreamID]
			if !isMapped {
				topicName = core.StreamRegistry.GetStreamName(msg.StreamID)
				prod.streamToTopic[msg.StreamID] = topicName
			}

			shared.Metric.New(kafkaMetricMessages + topicName)
			shared.Metric.New(kafkaMetricMessagesSec + topicName)
			prod.counters[topicName] = new(int64)

			topic, _ = kafka.NewTopic(topicName, kafka.NewRoundRobinPartitioner(), prod.topicConfig, prod.client)
			prod.topic[msg.StreamID] = topic
		}

		var key []byte
		if prod.keyFormat != nil {
			key, _ = prod.keyFormat.Format(msg)
		}
		kafkaMsg := kafka.NewSimpleMessage(key, msg.Data, originalMsg)

		if topicBatch, exists := batch[topic]; !exists {
			batch[topic] = []kafka.Message{kafkaMsg}
		} else {
			batch[topic] = append(topicBatch, kafkaMsg)
		}

		atomic.AddInt64(prod.counters[topic.GetName()], 1)
	}

	// Send messages
	for topic, messages := range batch {
		errors := topic.Produce(messages)
		for _, err := range errors {
			failed := err.Original.(kafka.SimpleMessage)
			failedMsg := failed.GetMetadata().(core.Message)
			Log.Error.Print(err)
			prod.Drop(failedMsg)
		}
	}
}

func (prod *KafkaProducer) isConnected() bool {
	return prod.client != nil
}

func (prod *KafkaProducer) tryConnect() bool {
	if prod.isConnected() {
		return true
	}

	client, err := kafka.NewProducer(prod.config)
	if err != nil {
		Log.Error.Print(err)
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

	client.Close()
	for _, topic := range topics {
		topic.Close()
	}
}

func (prod *KafkaProducer) close() {
	defer prod.WorkerDone()

	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
	prod.closeConnection()

	// TODO: Apparently those get freed by other means
	//prod.config.Close()
	//prod.topicConfig.Close()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *KafkaProducer) Produce(workers *sync.WaitGroup) {
	prod.tryConnect()
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
