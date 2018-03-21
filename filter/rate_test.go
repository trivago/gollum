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
	"testing"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFilterRate(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Rate")

	conf.Override("MessagesPerSec", 100)
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Rate)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, nil, 1)
	msg2 := core.NewMessage(nil, []byte{}, nil, 2)

	for i := 0; i < 110; i++ {
		result1, _ := filter.ApplyFilter(msg1)
		result2, _ := filter.ApplyFilter(msg2)
		if i < 100 {
			expect.Equal(core.FilterResultMessageAccept, result1)
			expect.Equal(core.FilterResultMessageAccept, result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.Neq(core.FilterResultMessageAccept, result1)
			expect.Neq(core.FilterResultMessageAccept, result2)
		}
	}

	// Wait for rates to be cleared
	time.Sleep(time.Second)

	for i := 0; i < 110; i++ {
		result1, _ := filter.ApplyFilter(msg1)
		result2, _ := filter.ApplyFilter(msg2)
		if i < 100 {
			expect.Equal(core.FilterResultMessageAccept, result1)
			expect.Equal(core.FilterResultMessageAccept, result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.Neq(core.FilterResultMessageAccept, result1)
			expect.Neq(core.FilterResultMessageAccept, result2)
		}
	}
}

func TestFilterRateIgnore(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Rate")

	conf.Override("MessagesPerSec", 100)
	conf.Override("Ignore", []string{core.LogInternalStream})
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Rate)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, nil, core.LogInternalStreamID)
	msg2 := core.NewMessage(nil, []byte{}, nil, 2)

	for i := 0; i < 200; i++ {
		result1, _ := filter.ApplyFilter(msg1)
		result2, _ := filter.ApplyFilter(msg2)

		expect.Equal(core.FilterResultMessageAccept, result1)
		if i < 100 {
			expect.Equal(core.FilterResultMessageAccept, result2)
			time.Sleep(time.Millisecond)
		} else {
			expect.Neq(core.FilterResultMessageAccept, result2)
		}
	}
}
