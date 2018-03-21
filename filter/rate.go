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

package filter

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/trivago/gollum/core"
)

// Rate filter plugin
//
// This plugin blocks messages after a certain number of messages per second
// has been reached.
//
// Parameters
//
// - MessagesPerSec: This value defines the maximum number of messages per second allowed
// to pass through this filter.
// By default this parameter is set to "100".
//
// - Ignore:  Defines a list of streams that should not be affected by
// rate limiting. This is useful for e.g. producers listeing to "*".
// By default this parameter is set to "empty".
//
// Examples
//
// This example accept ~10 messages in a second except the "noLimit" stream:
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - filter.Rate:
//        MessagesPerSec: 10
//        Ignore:
//          - noLimit
type Rate struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	stateGuard        *sync.RWMutex
	state             map[core.MessageStreamID]*rateState
	rateLimit         int64 `config:"MessagesPerSec" default:"100"`
	metricsRegistry   metrics.Registry
}

type rateState struct {
	lastReset   time.Time
	metricLimit metrics.Counter
	count       *int64
	ignore      bool
}

func init() {
	core.TypeRegistry.Register(Rate{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Rate) Configure(conf core.PluginConfigReader) {
	filter.stateGuard = new(sync.RWMutex)
	filter.state = make(map[core.MessageStreamID]*rateState)
	filter.metricsRegistry = core.NewMetricsRegistry("ratelimit")

	ignore := conf.GetStreamArray("Ignore", []core.MessageStreamID{})
	for _, stream := range ignore {
		filter.state[stream] = &rateState{
			ignore: true,
		}
	}
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *Rate) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	filter.stateGuard.RLock()
	state, known := filter.state[msg.GetStreamID()]
	filter.stateGuard.RUnlock()

	// Add stream if necessary
	if !known {
		filter.stateGuard.Lock()
		state = &rateState{
			count:       new(int64),
			ignore:      false,
			lastReset:   time.Now(),
			metricLimit: metrics.NewCounter(),
		}
		streamName := core.StreamRegistry.GetStreamName(msg.GetStreamID())
		filter.state[msg.GetStreamID()] = state
		filter.metricsRegistry.Register(streamName, state.metricLimit)
		filter.stateGuard.Unlock()
	}

	// Ignore if set
	if state.ignore {
		return core.FilterResultMessageAccept, nil // ### return, do not limit ###
	}

	// Reset if necessary
	if time.Since(state.lastReset) > time.Second {
		state.lastReset = time.Now()
		atomic.SwapInt64(state.count, 0)
	}

	// Check if to be filtered
	if atomic.AddInt64(state.count, 1) <= filter.rateLimit {
		return core.FilterResultMessageAccept, nil // ### return, do not limit ###
	}

	state.metricLimit.Inc(1)
	return filter.GetFilterResultMessageReject(), nil
}
