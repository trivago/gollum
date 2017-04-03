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

package filter

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
	"time"
)

func TestFilterRate(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("RateLimitPerSec", 100)
	plugin, err := core.NewPluginWithType("filter.Rate", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Rate)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, 0)
	msg2 := core.NewMessage(nil, []byte{}, 0)
	msg1.StreamID = 1
	msg2.StreamID = 2

	for i := 0; i < 110; i++ {
		result1 := filter.Accepts(msg1)
		result2 := filter.Accepts(msg2)
		if i < 100 {
			expect.True(result1)
			expect.True(result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.False(result1)
			expect.False(result2)
		}
	}

	// Wait for rates to be cleared
	time.Sleep(time.Second)

	for i := 0; i < 110; i++ {
		result1 := filter.Accepts(msg1)
		result2 := filter.Accepts(msg2)
		if i < 100 {
			expect.True(result1)
			expect.True(result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.False(result1)
			expect.False(result2)
		}
	}
}

func TestFilterRateIgnore(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("RateLimitPerSec", 100)
	conf.Override("RateLimitIgnore", []string{core.LogInternalStream})
	plugin, err := core.NewPluginWithType("filter.Rate", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Rate)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, 0)
	msg2 := core.NewMessage(nil, []byte{}, 0)
	msg1.StreamID = core.LogInternalStreamID
	msg2.StreamID = 2

	for i := 0; i < 200; i++ {
		result1 := filter.Accepts(msg1)
		result2 := filter.Accepts(msg2)

		expect.True(result1)
		if i < 100 {
			expect.True(result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.False(result2)
		}
	}
}
