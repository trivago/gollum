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
	"strconv"
	"sync"
	"time"

	"github.com/quipo/statsd"
	"github.com/trivago/gollum/core"
)

// StatsdMetrics producer
//
// This producer samples the messages it receives and sends metrics about them
// to statsd.
//
// Parameters
//
// - Server: Defines the server and port to send statsd metrics to.
// By default this parameter is set to "localhost:8125".
//
// - Prefix: Defines a string that is prepended to every statsd metric name.
// By default this parameter is set to "gollum.".
//
// - StreamMapping: Defines a translation from gollum stream to statsd metric
// name. If no mapping is given the gollum stream name is used as the metric
// name.
// By default this parameter is set to an empty list.
//
// - UseMessage: Switch between just counting all messages arriving at this
// producer or summing up the message content. If UseMessage is set to true, the
// contents will be parsed as an integer, i.e. a string containing a human
// readable number is expected.
// By default the parameter is set to false.
//
// - UseGauge: When set to true the statsd data format will switch from counter
// to gauge. Every stream that does not receive any message but is liste in
// StreamMapping will have a gauge value of 0.
// By default this is parameter is set to false.
//
// - Batch/MaxMessages: Defines the maximum number of messages to collect per
// batch.
// By default this parameter is set to 500.
//
// - Batch/TimeoutSec: Defines the number of seconds after which a batch is
// processed, regardless of MaxMessages being reached or not.
// By default this parameter is set to 10.
//
// Examples
//
// This example will collect all messages going through gollum and sending
// metrics about the different datastreams to statsd at least every 5 seconds.
// Metrics will be send as "logs.streamName".
//
//  metricsCollector:
//    Type: producer.StatsdMetrics
//    Stream: "*"
//    Server: "stats01:8125"
//    BatchTimeoutSec: 5
//    Prefix: "logs."
//    UseGauge: true
type StatsdMetrics struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	streamMap             map[core.MessageStreamID]string
	client                *statsd.StatsdClient
	batch                 core.MessageBatch
	flushFrequency        time.Duration `config:"Batch/TimeoutSec" default:"10" metric:"sec"`
	useMessage            bool          `config:"UseMessage" default:"false"`
	useGauge              bool          `config:"UseGauge" default:"false"`
}

func init() {
	core.TypeRegistry.Register(StatsdMetrics{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *StatsdMetrics) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(int(conf.GetInt("Batch/MaxMessages", 500)))

	server := conf.GetString("Server", "localhost:8125")
	prefix := conf.GetString("Prefix", "gollum.")
	prod.client = statsd.NewStatsdClient(server, prefix)
	conf.Errors.Push(prod.client.CreateSocket())
}

func (prod *StatsdMetrics) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *StatsdMetrics) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	} else if prod.useGauge && prod.batch.IsEmpty() {
		prod.transformMessages(make([]*core.Message, 0))
	}
}

func (prod *StatsdMetrics) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *StatsdMetrics) transformMessages(messages []*core.Message) {
	metricValues := make(map[string]int64)

	// Format and sort
	for _, msg := range messages {

		// Select the correct statsd metric
		metricName, streamMapped := prod.streamMap[msg.GetStreamID()]
		if !streamMapped {
			metricName, streamMapped = prod.streamMap[core.WildcardStreamID]
			if !streamMapped {
				metricName = core.StreamRegistry.GetStreamName(msg.GetStreamID())
				prod.streamMap[msg.GetStreamID()] = metricName
			}
		}

		_, metricMapped := metricValues[metricName]
		if !metricMapped {
			metricValues[metricName] = int64(0)
		}

		if prod.useMessage {
			// case msgData to int
			if val, err := strconv.ParseInt(msg.String(), 10, 64); err == nil {
				metricValues[metricName] += val
			} else {
				prod.Logger.Warning("message was skipped")
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

	// Send to StatsdMetrics
	for metric, val := range metricValues {
		if prod.useGauge {
			prod.client.Gauge(metric, val)
		} else {
			prod.client.Incr(metric, val)
		}
	}
}

func (prod *StatsdMetrics) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
	prod.client.Close()
}

// Produce writes to stdout or stderr.
func (prod *StatsdMetrics) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
