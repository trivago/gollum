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
	core.FilterBase
	stateGuard   *sync.RWMutex
	state        map[core.MessageStreamID]rateState
	dropStreamID core.MessageStreamID
	rateLimit    int64
}

type rateState struct {
	count  *int64
	ignore bool
}

func init() {
	core.TypeRegistry.Register(Rate{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Rate) Configure(conf core.PluginConfigReader) error {
	filter.FilterBase.Configure(conf)
	filter.rateLimit = int64(conf.GetInt("MessagesPerSec", 100))
	filter.stateGuard = new(sync.RWMutex)
	filter.state = make(map[core.MessageStreamID]rateState)

	dropToStream := conf.GetString("RateLimitDropToStream", "")
	if dropToStream != "" {
		filter.dropStreamID = core.GetStreamID(dropToStream)
	}

	ignore := conf.GetStreamArray("RateLimitIgnore", []core.MessageStreamID{})
	for _, stream := range ignore {
		filter.state[stream] = rateState{
			ignore: true,
		}
	}

	go filter.resetLoop()
	return conf.Errors.OrNil()
}

func (filter *Rate) resetLoop() {
	waitForReset := time.NewTicker(time.Second)
	for {
		<-waitForReset.C
		filter.stateGuard.RLock()
		for streamID, state := range filter.state {
			numFiltered := atomic.SwapInt64(state.count, 0)
			if numFiltered > filter.rateLimit {
				filter.Log.Warning.Printf("Ratelimit reached for stream %s: %d msg/sec", core.StreamRegistry.GetStreamName(streamID), numFiltered)
			}
		}
		filter.stateGuard.RUnlock()
	}
}

// Accepts allows all messages
func (filter *Rate) Accepts(msg *core.Message) bool {
	filter.stateGuard.RLock()
	state, known := filter.state[msg.StreamID]
	filter.stateGuard.RUnlock()

	// Add stream if necessary
	if !known {
		filter.stateGuard.Lock()
		state = rateState{
			count:  new(int64),
			ignore: false,
		}
		filter.state[msg.StreamID] = state
		filter.stateGuard.Unlock()
	} else if state.ignore {
		return true // ### return, do not limit ###
	}

	if atomic.AddInt64(state.count, 1) <= filter.rateLimit {
		return true
	}

	if filter.dropStreamID != core.InvalidStreamID {
		msg.Route(filter.dropStreamID)
	}

	return false

}
