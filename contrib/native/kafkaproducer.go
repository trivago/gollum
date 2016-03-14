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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type KafkaProducer struct {
	core.ProducerBase
	servers          []string
	topic            map[core.MessageStreamID]string
	clientID         string
	batch            core.MessageBatch
	counters         map[string]*int64
	lastMetricUpdate time.Time
	batchTimeout     time.Duration
	keyFormat        core.Formatter
	producer         libKafkaProducer
}

const (
	kafkaMetricMessages     = "Kafka:Messages-"
	kafkaMetricMessagesSec  = "Kafka:MessagesSec-"
	kafkaMetricUnresponsive = "Kafka:Unresponsive-"
	kafkaMetricMissCount    = "Kafka:ResponsesQueued"
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
	prod.topic = conf.GetStreamMap("Topic", "")
	prod.clientID = conf.GetString("ClientId", "gollum")
	prod.lastMetricUpdate = time.Now()
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second

	prod.batch = core.NewMessageBatch(conf.GetInt("Channel", 8192))
	prod.counters = make(map[string]*int64)

	for _, topic := range prod.topic {
		shared.Metric.New(kafkaMetricMessages + topic)
		shared.Metric.New(kafkaMetricMessagesSec + topic)
		shared.Metric.New(kafkaMetricUnresponsive + topic)
		prod.counters[topic] = new(int64)
	}

	shared.Metric.New(kafkaMetricMissCount)
	prod.SetCheckFuseCallback(prod.tryOpenConnection)

	compression := "none"
	switch strings.ToLower(conf.GetString("Compression", compressNone)) {
	case compressGZIP:
		compression = "gzip"
	case compressSnappy:
		compression = "snappy"
	}

	// Init librdkafka

	config := newLibKafkaConfig()
	if err := config.Set("compression.codec", compression); err != nil {
		return err
	}
	if err := config.SetI("batch.num.messages", prod.batch.Len()); err != nil {
		return err
	}
	if err := config.SetI("queue.buffering.max.ms", int(prod.batchTimeout*time.Millisecond)); err != nil {
		return err
	}
	if err := config.Set("client.id", conf.GetString("ClientId", "gollum")); err != nil {
		return err
	}
	if err := config.SetI("message.max.bytes", conf.GetInt("BatchSizeMaxKB", 1<<10)<<10); err != nil {
		return err
	}
	if err := config.SetI("topic.metadata.refresh.interval.ms", int(time.Duration(conf.GetInt("MetadataRefreshMs", 10000))*time.Millisecond)); err != nil {
		return err
	}
	if err := config.SetI("socket.timeout.ms", int(time.Duration(conf.GetInt("ServerTimeoutSec", 30))*time.Second*time.Millisecond)); err != nil {
		return err
	}
	if err := config.SetB("socket.keepalive.enables", true); err != nil {
		return err
	}

	prod.producer, err = newLibKafkaProducer(config)
	if err != nil {
		return err
	}

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
	if prod.tryOpenConnection() {
		prod.batch.Flush(prod.transformMessages)
	} else if prod.IsStopping() {
		prod.batch.Flush(prod.dropMessages)
	}
}

func (prod *KafkaProducer) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *KafkaProducer) transformMessages(messages []core.Message) {
	for _, msg := range messages {
		originalMsg := msg
		msg.Data, msg.StreamID = prod.ProducerBase.Format(msg)

		// Check if connected
		if !prod.isConnected() {
			prod.Drop(originalMsg)
			continue // ### return, not connected ###
		}

		// Send message
		topic, topicMapped := prod.topic[msg.StreamID]
		if !topicMapped {
			// Use wildcard fallback or stream name if not set
			topic, topicMapped = prod.topic[core.WildcardStreamID]
			if !topicMapped {
				topic = core.StreamRegistry.GetStreamName(msg.StreamID)
			}

			shared.Metric.New(kafkaMetricMessages + topic)
			shared.Metric.New(kafkaMetricMessagesSec + topic)
			shared.Metric.New(kafkaMetricUnresponsive + topic)
			prod.counters[topic] = new(int64)
			prod.topic[msg.StreamID] = topic
		}

		// TODO: native message
		// TODO: send
		/*
			kafkaMsg := &kafka.ProducerMessage{
				Topic:    topic,
				Value:    kafka.ByteEncoder(msg.Data),
				Metadata: originalMsg,
			}

			if prod.keyFormat != nil {
				key, _ := prod.keyFormat.Format(msg)
				kafkaMsg.Key = kafka.ByteEncoder(key)
			}
		*/
	}
}

func (prod *KafkaProducer) tryOpenConnection() bool {
	// TODO: native connect

	prod.Control() <- core.PluginControlFuseActive
	return true
}

func (prod *KafkaProducer) isConnected() bool {
	// TODO
	return true
}

func (prod *KafkaProducer) closeConnection() {
	// TODO: native disconnect
}

func (prod *KafkaProducer) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
	prod.closeConnection()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *KafkaProducer) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.tryOpenConnection()
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
