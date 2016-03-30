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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
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
//
// RateLimitPerSec defines the maximum number of messages per second allowed
// to pass through this filter. By default this is set to 100.
//
// RateLimitDropToStream is an optional stream messages are sent to when the
// limit is reached. By default this is disabled and set to "".
type Rate struct {
	counter      *int64
	rateLimit    int64
	dropStreamID core.MessageStreamID
}

func init() {
	shared.TypeRegistry.Register(Rate{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Rate) Configure(conf core.PluginConfig) error {
	filter.rateLimit = int64(conf.GetInt("RateLimitPerSec", 100))
	filter.dropStreamID = core.InvalidStreamID
	filter.counter = new(int64)

	dropToStream := conf.GetString("RateLimitDropToStream", "")
	if dropToStream != "" {
		filter.dropStreamID = core.GetStreamID(dropToStream)
	}
	go filter.resetLoop()
	return nil
}

func (filter *Rate) resetLoop() {
	waitForReset := time.NewTicker(time.Second)
	for {
		<-waitForReset.C
		numFiltered := atomic.SwapInt64(filter.counter, 0)
		if numFiltered > filter.rateLimit {
			Log.Warning.Printf("Ratelimit reached: %d msg/sec", numFiltered)
		}
	}
}

// Accepts allows all messages
func (filter *Rate) Accepts(msg core.Message) bool {
	if atomic.AddInt64(filter.counter, 1) > filter.rateLimit {
		if filter.dropStreamID != core.InvalidStreamID {
			msg.Route(filter.dropStreamID)
		}
		return false
	}
	return true
}
