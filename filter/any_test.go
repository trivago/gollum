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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFilterAnyAllNone(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Any")

	conf.Override("Any", []interface{}{"filter.None"})
	plugin, err := core.NewPlugin(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, 0, core.InvalidStreamID)

	result := filter.modulators[0].Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)
}

func TestFilterAnyJsonRegExp(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.Any")

	conf.Override("Any", []interface{}{
		"filter.JSON",
		map[interface{}]interface{}{
			"filter.RegExp": map[string]string{
				"Expression": "^ERROR",
			},
		},
	})
	plugin, err := core.NewPlugin(conf)
	expect.NoError(err)

	filter, casted := plugin.(*Any)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("ERROR"), 0, core.InvalidStreamID)

	result := filter.modulators[0].Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.modulators[1].Modulate(msg)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg)
	expect.Equal(core.ModulateResultContinue, result)

	msg = core.NewMessage(nil, []byte("{}"), 0, core.InvalidStreamID)

	result = filter.modulators[0].Modulate(msg)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.modulators[1].Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.Modulate(msg)
	expect.Equal(core.ModulateResultContinue, result)

	msg = core.NewMessage(nil, []byte("FAIL"), 0, core.InvalidStreamID)

	result = filter.modulators[0].Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.modulators[1].Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.Modulate(msg)
	expect.Equal(core.ModulateResultDiscard, result)
}
