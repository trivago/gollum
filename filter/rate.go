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

package filter

import (
	"github.com/trivago/gollum/core"
	"sync"
	"sync/atomic"
	"time"
)

// Rate filter plugin
// This plugin blocks messages after a certain number of messages per second
// has been reached.
// Configuration example
//
//   - "stream.Broadcast":
//     Filter: "filter.Rate"
//     RateLimitPerSec: 100
//     RateLimitDropToStream: ""
//     RateLimitIgnore:
//       - "foo"
//
// RateLimitPerSec defines the maximum number of messages per second allowed
// to pass through this filter. By default this is set to 100.
//
// RateLimitDropToStream is an optional stream messages are sent to when the
// limit is reached. By default this is disabled and set to "".
//
// RateLimitIgnore defines a list of streams that should not be affected by
// rate limiting. This is usefull for e.g. producers listeing to "*".
// By default this list is empty.
type Rate struct {
	core.SimpleFilter
	stateGuard *sync.RWMutex
	state      map[core.MessageStreamID]*rateState
	rateLimit  int64
}

type rateState struct {
	lastReset time.Time
	count     *int64
	ignore    bool
}

func init() {
	core.TypeRegistry.Register(Rate{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Rate) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)
	filter.rateLimit = int64(conf.GetInt("MessagesPerSec", 100))
	filter.stateGuard = new(sync.RWMutex)
	filter.state = make(map[core.MessageStreamID]*rateState)

	ignore := conf.GetStreamArray("Ignore", []core.MessageStreamID{})
	for _, stream := range ignore {
		filter.state[stream] = &rateState{
			count:     new(int64),
			ignore:    true,
			lastReset: time.Now().Add(-time.Second),
		}
	}

	return conf.Errors.OrNil()
}

// Accepts allows all messages
func (filter *Rate) Accepts(msg *core.Message) bool {
	filter.stateGuard.RLock()
	state, known := filter.state[msg.StreamID()]
	filter.stateGuard.RUnlock()

	// Add stream if necessary
	if !known {
		filter.stateGuard.Lock()
		state = &rateState{
			count:     new(int64),
			ignore:    false,
			lastReset: time.Now(),
		}
		filter.state[msg.StreamID()] = state
		filter.stateGuard.Unlock()
	}

	// Ignore if set
	if state.ignore {
		return true // ### return, do not limit ###
	}

	// Reset if necessary
	if time.Since(state.lastReset) > time.Second {
		state.lastReset = time.Now()
		numFiltered := atomic.SwapInt64(state.count, 0)
		if numFiltered > filter.rateLimit {
			filter.Log.Warning.Printf("Ratelimit reached for %s: %d msg/sec", core.StreamRegistry.GetStreamName(msg.StreamID()), numFiltered)
		}
	}

	// Check if to be filtered
	return atomic.AddInt64(state.count, 1) <= filter.rateLimit
}
