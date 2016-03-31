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
//
// RateLimitPerSec defines the maximum number of messages per second allowed
// to pass through this filter. By default this is set to 100.
//
// RateLimitDropToStream is an optional stream messages are sent to when the
// limit is reached. By default this is disabled and set to "".
type Rate struct {
	core.FilterBase
	counter   *int64
	rateLimit int64
}

func init() {
	core.TypeRegistry.Register(Rate{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Rate) Configure(conf core.PluginConfigReader) error {
	filter.FilterBase.Configure(conf)
	filter.rateLimit = int64(conf.GetInt("MessagesPerSec", 100))
	filter.counter = new(int64)

	go filter.resetLoop()
	return conf.Errors.OrNil()
}

func (filter *Rate) resetLoop() {
	waitForReset := time.NewTicker(time.Second)
	for {
		<-waitForReset.C
		numFiltered := atomic.SwapInt64(filter.counter, 0)
		if numFiltered > filter.rateLimit {
			filter.Log.Warning.Printf("Ratelimit reached: %d msg/sec", numFiltered)
		}
	}
}

// Accepts allows all messages
func (filter *Rate) Accepts(msg core.Message) bool {
	if atomic.AddInt64(filter.counter, 1) > filter.rateLimit {
		return false
	}
	return true
}
