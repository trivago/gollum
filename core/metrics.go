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

package core

import (
	"fmt"
	"github.com/trivago/tgo"
	"time"
)

const (
	metricRouters         = "Routers"
	metricFallbackRouters = "Routers:Fallback"
	metricCons            = "Consumers"
	metricProds           = "Producers"
	metricVersion         = "Version"

	metricMessagesRouted = "Messages:Routed"
	// MetricMessagesRoutedAvg is used as a key for storing message throughput
	MetricMessagesRoutedAvg    = "Messages:Routed:AvgPerSec"
	metricMessagesEnqued       = "Messages:Enqueued"
	metricMessagesEnquedAvg    = "Messages:Enqueued:AvgPerSec"
	metricMessagesDiscarded    = "Messages:Discarded"
	metricMessagesDiscardedSec = "Messages:Discarded:AvgPerSec"

	metricStreamMessagesRouted       = "Stream:%s:Messages:Routed"
	metricStreamMessagesRoutedAvg    = "Stream:%s:Messages:Routed:AvgPerSec"
	metricStreamMessagesDiscarded    = "Stream:%s:Messages:Discarded"
	metricStreamMessagesDiscardedAvg = "Stream:%s:Messages:Discarded:AvgPerSec"
)

const (
	MetricActiveWorkers      = "Plugins:ActiveWorkers"
	MetricPluginsInit        = "Plugins:State:Initializing"
	MetricPluginsActive      = "Plugins:State:Active"
	MetricPluginsWaiting     = "Plugins:State:Waiting"
	MetricPluginsPrepareStop = "Plugins:State:PrepareStop"
	MetricPluginsStopping    = "Plugins:State:Stopping"
	MetricPluginsDead        = "Plugins:State:Dead"
)

func init() {
	tgo.EnableGlobalMetrics()
	tgo.Metric.InitSystemMetrics()

	tgo.Metric.New(metricVersion)
	tgo.Metric.Set(metricVersion, GetVersionNumber())

	tgo.Metric.New(metricRouters)
	tgo.Metric.New(metricFallbackRouters)
	tgo.Metric.New(metricCons)
	tgo.Metric.New(metricProds)

	tgo.Metric.New(MetricPluginsInit)
	tgo.Metric.New(MetricPluginsWaiting)
	tgo.Metric.New(MetricPluginsActive)
	tgo.Metric.New(MetricPluginsPrepareStop)
	tgo.Metric.New(MetricPluginsStopping)
	tgo.Metric.New(MetricPluginsDead)
	tgo.Metric.New(MetricActiveWorkers)

	tgo.Metric.New(metricMessagesRouted)
	tgo.Metric.New(metricMessagesEnqued)
	tgo.Metric.New(metricMessagesDiscarded)
	tgo.Metric.NewRate(metricMessagesRouted, MetricMessagesRoutedAvg, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricMessagesEnqued, metricMessagesEnquedAvg, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricMessagesDiscarded, metricMessagesDiscardedSec, time.Second, 10, 3, true)
}

// CountMessageRouted increases the messages counter by 1
func CountMessageRouted() {
	tgo.Metric.Inc(metricMessagesRouted)
}

// CountMessageDiscarded increases the discarded messages counter by 1
func CountMessageDiscarded() {
	tgo.Metric.Inc(metricMessagesDiscarded)
}

// CountMessagesEnqueued increases the enqueued messages counter by 1
func CountMessagesEnqueued() {
	tgo.Metric.Inc(metricMessagesEnqued)
}

// CountProducers increases the producer counter by 1
func CountProducers() {
	tgo.Metric.Inc(metricProds)
}

// CountConsumers increases the consumer counter by 1
func CountConsumers() {
	tgo.Metric.Inc(metricCons)
}

// CountRouters increases the stream counter by 1
func CountRouters() {
	tgo.Metric.Inc(metricRouters)
}

// CountFallbackRouters increases the fallback stream counter by 1
func CountFallbackRouters() {
	tgo.Metric.Inc(metricFallbackRouters)
}

var streamMetrics = map[MessageStreamID]StreamMetric{}

// GetSteamMetric return a StreamMetric instance for the given MessageStreamID
func GetSteamMetric(streamID MessageStreamID) StreamMetric {
	if metric, isSet := streamMetrics[streamID]; isSet {
		return metric
	}

	metric := StreamMetric{
		streamID,
	}
	metric.init()
	streamMetrics[streamID] = metric

	return metric
}

// StreamMetric class for stream based metrics
type StreamMetric struct {
	messageStreamID MessageStreamID
}

func (metric *StreamMetric) init() {
	keyRouted := metric.getMetricKey(metricStreamMessagesRouted)
	keyDiscarded := metric.getMetricKey(metricStreamMessagesDiscarded)

	tgo.Metric.New(keyRouted)
	tgo.Metric.New(keyDiscarded)

	tgo.Metric.NewRate(keyRouted, metric.getMetricKey(metricStreamMessagesRoutedAvg), time.Second, 10, 3, true)
	tgo.Metric.NewRate(keyDiscarded, metric.getMetricKey(metricStreamMessagesDiscardedAvg), time.Second, 10, 3, true)
}

// CountMessageRouted increases the messages counter by 1
func (metric *StreamMetric) CountMessageRouted() {
	tgo.Metric.Inc(metric.getMetricKey(metricStreamMessagesRouted))
}

// CountMessageDiscarded increases the discarded messages counter by 1
func (metric *StreamMetric) CountMessageDiscarded() {
	tgo.Metric.Inc(metric.getMetricKey(metricStreamMessagesDiscarded))
}

func (metric *StreamMetric) getMetricKey(format string) string {
	return fmt.Sprintf(format, StreamRegistry.GetStreamName(metric.messageStreamID))
}

// PluginMetric class for plugin based metrics
type PluginMetric struct {

}

// Init increase the plugin state init count
func (metric *PluginMetric) Init() {
	tgo.Metric.Inc(MetricPluginsInit)
}

// UpdateStateMetric decrease the prev plugin state and increase the next plugin state
func (metric *PluginMetric) UpdateStateMetric(prevState, nextState string) {
	tgo.Metric.Dec(prevState)
	tgo.Metric.Inc(nextState)
}

// IncWorker increase the active worker count
func (metric *PluginMetric) IncWorker() {
	tgo.Metric.Inc(MetricActiveWorkers)
}

// DecWorker decrease the active worker count
func (metric *PluginMetric) DecWorker() {
	tgo.Metric.Dec(MetricActiveWorkers)
}