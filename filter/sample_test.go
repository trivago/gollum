// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	 http://www.apache.org/licenses/LICENSE-2.0
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
)

func TestFilterSample(t *testing.T) {
	expect := shared.NewExpect(t)
	msg := core.NewMessage(nil, []byte{}, 0)
	msg.StreamID = 1

	conf := core.NewPluginConfig("")
	conf.Override("SampleRatePerGroup", 2)
	conf.Override("SampleGroupSize", 5)
	plugin, err := core.NewPluginWithType("filter.Sample", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Sample)
	expect.True(casted)

	accept, deny := 0, 0
	for i := 0; i < 10; i++ {
		result := filter.Accepts(msg)
		if result {
			accept += 1
		} else {
			deny += 1
		}
	}
	expect.Equal(accept, 4)
	expect.Equal(deny, 6)

	conf = core.NewPluginConfig("")
	conf.Override("SampleGroupSize", 2)
	plugin, err = core.NewPluginWithType("filter.Sample", conf)
	expect.NoError(err)

	filter, casted = plugin.(*Sample)
	expect.True(casted)

	accept, deny = 0, 0
	for i := 0; i < 10; i++ {
		result := filter.Accepts(msg)
		if result {
			accept += 1
		} else {
			deny += 1
		}
	}
	expect.Equal(accept, 5)
	expect.Equal(deny, 5)
}

func TestFilterSampleIgnore(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("SampleGroupSize", 2)
	conf.Override("SampleIgnore", []string{core.LogInternalStream})
	plugin, err := core.NewPluginWithType("filter.Sample", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Sample)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, 0)
	msg2 := core.NewMessage(nil, []byte{}, 0)
	msg1.StreamID = core.LogInternalStreamID
	msg2.StreamID = 2

	accept1, accept2, deny1, deny2 := 0, 0, 0, 0
	for i := 0; i < 10; i++ {
		result1 := filter.Accepts(msg1)
		result2 := filter.Accepts(msg2)

		if result1 {
			accept1 += 1
		} else {
			deny1 += 1
		}
		if result2 {
			accept2 += 1
		} else {
			deny2 += 1
		}
	}
	expect.Equal(accept1, 10)
	expect.Equal(deny1, 0)
	expect.Equal(accept2, 5)
	expect.Equal(deny2, 5)
}
