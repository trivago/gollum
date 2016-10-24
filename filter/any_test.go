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

func TestFilterAnyAllNone(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("AnyFilter", []string{"filter.None", "filter.All"})
	plugin, err := core.NewPluginWithType("filter.Any", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, 0)
	expect.False(filter.filters[0].Accepts(msg))
	expect.True(filter.filters[1].Accepts(msg))
	expect.True(filter.Accepts(msg))
}

func TestFilterAnyJsonRegExp(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("AnyFilter", []string{"filter.JSON", "filter.RegExp"})
	conf.Override("FilterExpression", "^ERROR")
	plugin, err := core.NewPluginWithType("filter.Any", conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("ERROR"), 0)
	expect.False(filter.filters[0].Accepts(msg))
	expect.True(filter.filters[1].Accepts(msg))
	expect.True(filter.Accepts(msg))

	msg = core.NewMessage(nil, []byte("{}"), 0)
	expect.True(filter.filters[0].Accepts(msg))
	expect.False(filter.filters[1].Accepts(msg))
	expect.True(filter.Accepts(msg))

	msg = core.NewMessage(nil, []byte("FAIL"), 0)
	expect.False(filter.filters[0].Accepts(msg))
	expect.False(filter.filters[1].Accepts(msg))
	expect.False(filter.Accepts(msg))
}
