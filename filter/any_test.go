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

func TestFilterAnyAllNone(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Any")

	conf.Override("AnyFilters", []interface{}{"filter.None"})
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, nil, core.InvalidStreamID)

	result, err := filter.filters[0].ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)
	expect.NoError(err)

	result, err = filter.ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)
	expect.NoError(err)
}

func TestFilterAnyMulti(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Any")

	conf.Override("AnyFilters", []interface{}{
		map[interface{}]interface{}{
			"filter.RegExp": map[string]string{
				"Expression": "[{}]+",
			},
		},
		map[interface{}]interface{}{
			"filter.RegExp": map[string]string{
				"Expression": "^ERROR",
			},
		},
	})
	plugin, err := core.NewPluginWithConfig(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	// test case 1
	msg := core.NewMessage(nil, []byte("ERROR"), nil, core.InvalidStreamID)

	result, _ := filter.filters[0].ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.filters[1].ApplyFilter(msg)
	expect.Equal(core.FilterResultMessageAccept, result)

	result, err = filter.ApplyFilter(msg)
	expect.Equal(core.FilterResultMessageAccept, result)
	expect.NoError(err)

	// test case 2
	msg = core.NewMessage(nil, []byte("{}"), nil, core.InvalidStreamID)

	result, _ = filter.filters[0].ApplyFilter(msg)
	expect.Equal(core.FilterResultMessageAccept, result)

	result, _ = filter.filters[1].ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.ApplyFilter(msg)
	expect.Equal(core.FilterResultMessageAccept, result)

	// test case 3
	msg = core.NewMessage(nil, []byte("FAIL"), nil, core.InvalidStreamID)

	result, _ = filter.filters[0].ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.filters[1].ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)

	result, _ = filter.ApplyFilter(msg)
	expect.Neq(core.FilterResultMessageAccept, result)
}
