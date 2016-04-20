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
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestFilterStream(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	plugin, err := core.NewPluginWithType("filter.Stream", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Stream)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, 0)
	msg1.StreamID = 1

	msg2 := core.NewMessage(nil, []byte{}, 0)
	msg2.StreamID = 2

	filter.blacklist = []core.MessageStreamID{1}

	expect.False(filter.Accepts(msg1))
	expect.True(filter.Accepts(msg2))

	filter.whitelist = []core.MessageStreamID{1}

	expect.False(filter.Accepts(msg1))
	expect.False(filter.Accepts(msg2))

	filter.blacklist = []core.MessageStreamID{}

	expect.True(filter.Accepts(msg1))
	expect.False(filter.Accepts(msg2))
}
