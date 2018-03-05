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

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFilterStream(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Stream")

	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Stream)
	expect.True(casted)

	msg1 := core.NewMessage(nil, []byte{}, nil, 1)
	msg2 := core.NewMessage(nil, []byte{}, nil, 2)

	filter.blacklist = []core.MessageStreamID{1}

	result, _ := filter.ApplyFilter(msg1)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.ApplyFilter(msg2)
	expect.Equal(core.FilterResultMessageAccept, result)

	filter.whitelist = []core.MessageStreamID{1}

	result, _ = filter.ApplyFilter(msg1)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.ApplyFilter(msg2)
	expect.Neq(core.FilterResultMessageAccept, result)

	filter.blacklist = []core.MessageStreamID{}

	result, _ = filter.ApplyFilter(msg1)
	expect.Equal(core.FilterResultMessageAccept, result)

	result, _ = filter.ApplyFilter(msg2)
	expect.Neq(core.FilterResultMessageAccept, result)
}
