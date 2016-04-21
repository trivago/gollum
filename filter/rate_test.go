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
	"github.com/trivago/tgo/ttesting"
	"testing"
	"time"
)

func TestFilterRate(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Rate")

	conf.Override("RateLimitPerSec", 100)
	plugin, err := core.NewPlugin(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Rate)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, 0)

	for i := 0; i < 110; i++ {
		result := filter.Accepts(msg)
		if i < 100 {
			expect.True(result)
			time.Sleep(time.Millisecond)
		} else {
			expect.False(result)
		}
	}

	time.Sleep(time.Second)

	for i := 0; i < 110; i++ {
		result := filter.Accepts(msg)
		if i < 100 {
			expect.True(result)
			time.Sleep(time.Millisecond)
		} else {
			expect.False(result)
		}
	}
}
