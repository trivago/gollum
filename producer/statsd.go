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
	"github.com/quipo/statsd"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strconv"
	"sync"
	"time"
)

// Statsd producer plugin
// This producer sends increment events to a statsd server.
// Configuration example
//
//  - "producer.Statsd":
//    BatchMaxMessages: 500
//    BatchTimeoutSec: 10
//    Prefix: "gollum."
//    Server: "localhost:8125"
//    UseMessage: false
//    UseGauge: false
//    StreamMapping:
//      "*" : "default"
//
// BatchMaxMessages defines the maximum number of messages to send per
// batch. By default this is set to 500.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 10.
//
// Prefix defines the prefix for stats metric names. By default this
// is set to "gollum.".
//
// Server defines the server and port to send statsd metrics to. By default
// this is set to "localhost:8125".
//
// UseMessage defines whether to cast the message to string and increment
// the metric by that value. If this is set to true and the message fails
// to cast to an integer, then the message with be ignored. If this is set
// to false then each message will increment by 1. By default this is set
// to false.
//
// UseGauge defines whether to send gauge metrics instead of counter
// metrics. If this is set to true then every stream in streamMap that
// does not receive any messages in a batch will have a gauge value of 0
// sent for that batch. By default this is set to false.
//
// StreamMapping defines a translation from gollum stream to statsd metric
// name. If no mapping is given the gollum stream name is used as the
// metric name.
type Statsd struct {
	core.ProducerBase
	streamMap      map[core.MessageStreamID]string
	client         *statsd.StatsdClient
	batch          core.MessageBatch
	flushFrequency time.Duration
	useMessage     bool
	useGauge       bool
}

func init() {
	shared.TypeRegistry.Register(Statsd{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Statsd) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 500))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 10)) * time.Second
	prod.useMessage = conf.GetBool("UseMessage", false)
	prod.useGauge = conf.GetBool("UseGauge", false)

	server := conf.GetString("Server", "localhost:8125")
	prefix := conf.GetString("Prefix", "gollum.")
	prod.client = statsd.NewStatsdClient(server, prefix)
	err = prod.client.CreateSocket()
	if nil != err {
		return err
	}

	return nil
}

func (prod *Statsd) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *Statsd) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	} else if prod.useGauge && prod.batch.IsEmpty() {
		prod.transformMessages(make([]core.Message, 0))
	}
}

func (prod *Statsd) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Statsd) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Statsd) transformMessages(messages []core.Message) {
	metricValues := make(map[string]int64)

	// Format and sort
	for _, msg := range messages {
		msgData, streamID := prod.ProducerBase.Format(msg)

		// Select the correct statsd metric
		metricName, streamMapped := prod.streamMap[streamID]
		if !streamMapped {
			metricName, streamMapped = prod.streamMap[core.WildcardStreamID]
			if !streamMapped {
				metricName = core.StreamRegistry.GetStreamName(streamID)
				prod.streamMap[streamID] = metricName
			}
		}

		_, metricMapped := metricValues[metricName]
		if !metricMapped {
			metricValues[metricName] = int64(0)
		}

		if prod.useMessage {
			// case msgData to int
			if val, err := strconv.ParseInt(string(msgData), 10, 64); err == nil {
				metricValues[metricName] += val
			} else {
				Log.Warning.Print("producer.Statsd: message was skipped")
			}
		} else {
			metricValues[metricName] += int64(1)
		}
	}

	if prod.useGauge {
		// add a 0 for all mapped streams with no value
		for _, metricName := range prod.streamMap {
			if _, metricMapped := metricValues[metricName]; !metricMapped {
				metricValues[metricName] = int64(0)
			}
		}
	}

	// Send to Statsd
	for metric, val := range metricValues {
		if prod.useGauge {
			prod.client.Gauge(metric, val)
		} else {
			prod.client.Incr(metric, val)
		}
	}
}

func (prod *Statsd) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
	prod.client.Close()
}

// Produce writes to stdout or stderr.
func (prod *Statsd) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
