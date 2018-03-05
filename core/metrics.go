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

package core

import (
	"fmt"
	"github.com/trivago/tgo"
	"sync"
	"time"
)

const (
	metricRouters         = "Routers"
	metricFallbackRouters = "Routers:Fallback"
	metricCons            = "Consumers"
	metricProds           = "Producers"
	metricVersion         = "Version"
)

// MetricMessagesRoutedAvg is used as a key for storing message throughput
const (
	metricMessagesRouted       = "Messages:Routed"
	MetricMessagesRoutedAvg    = "Messages:Routed:AvgPerSec"
	metricMessagesEnqued       = "Messages:Enqueued"
	metricMessagesEnquedAvg    = "Messages:Enqueued:AvgPerSec"
	metricMessagesDiscarded    = "Messages:Discarded"
	metricMessagesDiscardedSec = "Messages:Discarded:AvgPerSec"
)

const (
	metricStreamMessagesRouted       = "Stream:%s:Messages:Routed"
	metricStreamMessagesRoutedAvg    = "Stream:%s:Messages:Routed:AvgPerSec"
	metricStreamMessagesDiscarded    = "Stream:%s:Messages:Discarded"
	metricStreamMessagesDiscardedAvg = "Stream:%s:Messages:Discarded:AvgPerSec"
)

// MetricActiveWorkers metric string
// MetricPluginsInit metric string
// MetricPluginsActive metric string
// MetricPluginsWaiting metric string
// MetricPluginsPrepareStop metric string
// MetricPluginsStopping metric string
// MetricPluginsDead metric string
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
var streamMetricsGuard = new(sync.Mutex)

// GetStreamMetric returns a StreamMetric instance for the given MessageStreamID
func GetStreamMetric(streamID MessageStreamID) StreamMetric {
	streamMetricsGuard.Lock()
	metric, isSet := streamMetrics[streamID]
	streamMetricsGuard.Unlock()

	if isSet {
		return metric
	}

	metric = newStreamMetric(streamID)

	streamMetricsGuard.Lock()
	if racedMetric, isSet := streamMetrics[streamID]; isSet {
		metric = racedMetric
	} else {
		streamMetrics[streamID] = metric
	}
	streamMetricsGuard.Unlock()

	return metric
}

// StreamMetric class for stream based metrics
type StreamMetric struct {
	keyRouted    string
	keyDiscarded string
}

func newStreamMetric(streamID MessageStreamID) StreamMetric {
	streamName := StreamRegistry.GetStreamName(streamID)

	metric := StreamMetric{
		keyRouted:    fmt.Sprintf(metricStreamMessagesRouted, streamName),
		keyDiscarded: fmt.Sprintf(metricStreamMessagesDiscarded, streamName),
	}

	keyRoutedAvg := fmt.Sprintf(metricStreamMessagesRoutedAvg, streamName)
	keyDiscardedAvg := fmt.Sprintf(metricStreamMessagesDiscardedAvg, streamName)

	tgo.Metric.New(metric.keyRouted)
	tgo.Metric.New(metric.keyDiscarded)
	tgo.Metric.NewRate(metric.keyRouted, keyRoutedAvg, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metric.keyDiscarded, keyDiscardedAvg, time.Second, 10, 3, true)

	return metric
}

// CountMessageRouted increases the messages counter by 1
func (metric *StreamMetric) CountMessageRouted() {
	tgo.Metric.Inc(metric.keyRouted)
}

// CountMessageDiscarded increases the discarded messages counter by 1
func (metric *StreamMetric) CountMessageDiscarded() {
	tgo.Metric.Inc(metric.keyDiscarded)
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
