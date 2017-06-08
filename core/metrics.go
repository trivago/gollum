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
	metricStreams        = "Streams"
	metricMessagesRouted = "Messages:Routed"
	// MetricMessagesSec is used as a key for storing message throughput
	MetricMessagesSec       = "Messages:Routed:AvgPerSec"
	metricMessagesEnqued    = "Messages:Enqueued"
	metricMessagesEnquedAvg = "Messages:Enqueued:AvgPerSec"
	metricDiscarded         = "Messages:Discarded"
	metricDiscardedSec      = "Messages:Discarded:AvgPerSec"

	metricStreamMessagesRouted    = "Stream:%s:Messages:Routed"
	metricStreamMessagesRoutedAvg = "Stream:%s:Messages:Routed:AvgPerSec"
	metricStreamDiscarded         = "Stream:%s:Messages:Discarded"
	metricStreamDiscardedAvg      = "Stream:%s:Messages:Discarded:AvgPerSec"
)

func init() {
	tgo.EnableGlobalMetrics()

	tgo.Metric.New(metricStreams)
	tgo.Metric.New(metricMessagesRouted)
	tgo.Metric.New(metricMessagesEnqued)
	tgo.Metric.New(metricDiscarded)
	tgo.Metric.NewRate(metricMessagesRouted, MetricMessagesSec, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricMessagesEnqued, metricMessagesEnquedAvg, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricDiscarded, metricDiscardedSec, time.Second, 10, 3, true)
}

// CountMessageRouted increases the messages counter by 1
func CountMessageRouted() {
	tgo.Metric.Inc(metricMessagesRouted)
}

// CountMessageDiscarded increases the discarded messages counter by 1
func CountMessageDiscarded() {
	tgo.Metric.Inc(metricDiscarded)
}

// CountMessagesEnqueued increases the enqueued messages counter by 1
func CountMessagesEnqueued() {
	tgo.Metric.Inc(metricMessagesEnqued)
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
	keyDiscarded := metric.getMetricKey(metricStreamDiscarded)

	tgo.Metric.New(keyRouted)
	tgo.Metric.New(keyDiscarded)

	tgo.Metric.NewRate(keyRouted, metric.getMetricKey(metricStreamMessagesRoutedAvg), time.Second, 10, 3, true)
	tgo.Metric.NewRate(keyDiscarded, metric.getMetricKey(metricStreamDiscardedAvg), time.Second, 10, 3, true)
}

// CountMessageRouted increases the messages counter by 1
func (metric *StreamMetric) CountMessageRouted() {
	tgo.Metric.Inc(metric.getMetricKey(metricStreamMessagesRouted))
}

// CountMessageDiscarded increases the discarded messages counter by 1
func (metric *StreamMetric) CountMessageDiscarded() {
	tgo.Metric.Inc(metric.getMetricKey(metricStreamDiscarded))
}

func (metric *StreamMetric) getMetricKey(format string) string {
	return fmt.Sprintf(format, StreamRegistry.GetStreamName(metric.messageStreamID))
}
