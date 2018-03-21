// Copyright 2015-2018 trivago N.V.
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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFilterSample(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msg := core.NewMessage(nil, []byte{}, nil, 1)

	conf := core.NewPluginConfig("", "filter.Sample")
	conf.Override("SampleRatePerGroup", uint64(2))
	conf.Override("SampleGroupSize", uint64(5))
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Sample)
	expect.True(casted)

	accept, deny := 0, 0
	for i := 0; i < 10; i++ {
		result, _ := filter.ApplyFilter(msg)
		if result == core.FilterResultMessageAccept {
			accept++
		} else {
			deny++
		}
	}
	expect.Equal(accept, 4)
	expect.Equal(deny, 6)

	conf = core.NewPluginConfig("", "filter.Sample")
	conf.Override("SampleGroupSize", uint64(2))
	plugin, err = core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted = plugin.(*Sample)
	expect.True(casted)

	accept, deny = 0, 0
	for i := 0; i < 10; i++ {
		result, _ := filter.ApplyFilter(msg)
		if result == core.FilterResultMessageAccept {
			accept++
		} else {
			deny++
		}
	}
	expect.Equal(accept, 5)
	expect.Equal(deny, 5)
}

func TestFilterSampleIgnore(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Sample")

	conf.Override("SampleGroupSize", uint64(2))
	conf.Override("SampleIgnore", []string{core.LogInternalStream})
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Sample)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, nil, core.LogInternalStreamID)
	msg2 := core.NewMessage(nil, []byte{}, nil, 2)

	accept1, accept2, deny1, deny2 := 0, 0, 0, 0
	for i := 0; i < 10; i++ {
		result1, _ := filter.ApplyFilter(msg1)
		result2, _ := filter.ApplyFilter(msg2)

		if result1 == core.FilterResultMessageAccept {
			accept1++
		} else {
			deny1++
		}
		if result2 == core.FilterResultMessageAccept {
			accept2++
		} else {
			deny2++
		}
	}
	expect.Equal(accept1, 10)
	expect.Equal(deny1, 0)
	expect.Equal(accept2, 5)
	expect.Equal(deny2, 5)
}
