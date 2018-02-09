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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/trivago/tgo/tmath"
)

// StreamMetric holds per-stream metrics objects
type StreamMetric struct {
	registry  metrics.Registry
	Routed    metrics.Counter
	Discarded metrics.Counter
}

var (
	// MetricsRegistry is the root registry for all metrics
	MetricsRegistry            metrics.Registry
	pluginMetricsRegistry      metrics.Registry
	metricsStreamRegistry      map[MessageStreamID]*StreamMetric
	metricsStreamRegistryGuard sync.RWMutex

	metricVersion   metrics.Gauge
	metricGoVersion metrics.Gauge

	// MetricRouters holds the total number of routers created
	MetricRouters metrics.Counter
	// MetricFallbackRouters holds the total number of fallback routers created
	MetricFallbackRouters metrics.Counter
	// MetricConsumers holds the total number of consumer created
	MetricConsumers metrics.Counter
	// MetricProducers holds the total number of producers created
	MetricProducers metrics.Counter
	// MetricActiveWorkers holds the number of currently active workers
	MetricActiveWorkers metrics.Gauge
	// MetricPluginsInit holds the number of plugins in the init state
	MetricPluginsInit metrics.Gauge
	// MetricPluginsWaiting holds the number of plugins in the waiting state
	MetricPluginsWaiting metrics.Gauge
	// MetricPluginsActive holds the number of plugins in the active state
	MetricPluginsActive metrics.Gauge
	// MetricPluginsPrepareStop holds the number of plugins in the prepare stop state
	MetricPluginsPrepareStop metrics.Gauge
	// MetricPluginsStopping holds the number of plugins in the stopping state
	MetricPluginsStopping metrics.Gauge
	// MetricPluginsDead holds the number of plugins in the dead state
	MetricPluginsDead metrics.Gauge
	// MetricMessagesRouted holds the total number of routed messages
	MetricMessagesRouted metrics.Counter
	// MetricMessagesEnqued holds the total number of enqueued messages
	MetricMessagesEnqued metrics.Counter
	// MetricMessagesDiscarded holds the total number of discarded messages
	MetricMessagesDiscarded metrics.Counter
)

func init() {
	MetricsRegistry = metrics.NewRegistry()
	metricsStreamRegistry = make(map[MessageStreamID]*StreamMetric)

	metricVersion = metrics.NewRegisteredGauge("version", MetricsRegistry)
	metricGoVersion = metrics.NewRegisteredGauge("go_version", MetricsRegistry)
	MetricMessagesRouted = metrics.NewRegisteredCounter("routed", MetricsRegistry)
	MetricMessagesEnqued = metrics.NewRegisteredCounter("enqueued", MetricsRegistry)
	MetricMessagesDiscarded = metrics.NewRegisteredCounter("discarded", MetricsRegistry)
	MetricActiveWorkers = metrics.NewRegisteredGauge("workers", MetricsRegistry)

	pluginMetricsRegistry = NewSubRegistry("plugins")
	MetricRouters = metrics.NewRegisteredCounter("routers", pluginMetricsRegistry)
	MetricFallbackRouters = metrics.NewRegisteredCounter("routers_default", pluginMetricsRegistry)
	MetricConsumers = metrics.NewRegisteredCounter("consumers", pluginMetricsRegistry)
	MetricProducers = metrics.NewRegisteredCounter("producers", pluginMetricsRegistry)
	MetricPluginsInit = metrics.NewRegisteredGauge("init", pluginMetricsRegistry)
	MetricPluginsWaiting = metrics.NewRegisteredGauge("waiting", pluginMetricsRegistry)
	MetricPluginsActive = metrics.NewRegisteredGauge("active", pluginMetricsRegistry)
	MetricPluginsPrepareStop = metrics.NewRegisteredGauge("preparestop", pluginMetricsRegistry)
	MetricPluginsStopping = metrics.NewRegisteredGauge("stopping", pluginMetricsRegistry)
	MetricPluginsDead = metrics.NewRegisteredGauge("dead", pluginMetricsRegistry)

	stateToMetric[PluginStateInitializing] = MetricPluginsInit
	stateToMetric[PluginStateWaiting] = MetricPluginsWaiting
	stateToMetric[PluginStateActive] = MetricPluginsActive
	stateToMetric[PluginStatePrepareStop] = MetricPluginsPrepareStop
	stateToMetric[PluginStateStopping] = MetricPluginsStopping
	stateToMetric[PluginStateDead] = MetricPluginsDead

	metrics.RegisterRuntimeMemStats(MetricsRegistry)

	// Populate constant values

	version := runtime.Version()
	if version[0] == 'g' && version[1] == 'o' {
		parts := strings.Split(version[2:], ".")
		numericVersion := make([]uint64, tmath.MaxI(3, len(parts)))
		for i, p := range parts {
			numericVersion[i], _ = strconv.ParseUint(p, 10, 64)
		}
		metricGoVersion.Update(int64(numericVersion[0]*10000 + numericVersion[1]*100 + numericVersion[2]))
	}

	metricGoVersion.Update(GetVersionNumber())
	go metrics.CaptureRuntimeMemStats(MetricsRegistry, time.Second)
}

// NewSubRegistry creates a new, prefixed metrics registry that can be used
// to register custom plugin metrics.
func NewSubRegistry(prefix string) metrics.Registry {
	return metrics.NewPrefixedChildRegistry(MetricsRegistry, prefix)
}

// NewPluginRegistry calls NewSubRegistry witht he id of the given plugin.
func NewPluginRegistry(plugin PluginWithID) metrics.Registry {
	return NewSubRegistry(plugin.GetID())
}

// GetStreamMetric returns the metrics handles for a given stream.
func GetStreamMetric(streamID MessageStreamID) *StreamMetric {
	metricsStreamRegistryGuard.RLock()
	if stream, ok := metricsStreamRegistry[streamID]; ok {
		metricsStreamRegistryGuard.RUnlock()
		return stream
	}
	metricsStreamRegistryGuard.RUnlock()

	metricsStreamRegistryGuard.Lock()
	defer metricsStreamRegistryGuard.Unlock()

	stream := &StreamMetric{
		registry:  NewSubRegistry(streamID.GetName()),
		Routed:    metrics.NewCounter(),
		Discarded: metrics.NewCounter(),
	}

	stream.registry.Register("routed", stream.Routed)
	stream.registry.Register("discarded", stream.Discarded)
	metricsStreamRegistry[streamID] = stream

	return stream
}
