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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFilterRegExp(t *testing.T) {
	expect := ttesting.NewExpect(t)
	conf := core.NewPluginConfig("", "filter.RegExp")

	conf.Override("ExpressionNot", "^\\d")
	conf.Override("Expression", "accept")
	plugin, err := core.NewPlugin(conf)
	expect.NoError(err)

	filter, casted := plugin.(*RegExp)
	expect.True(casted)

	msg1 := core.NewMessage(nil, ([]byte)("accept"), 0, core.InvalidStreamID)
	msg2 := core.NewMessage(nil, ([]byte)("0accept"), 0, core.InvalidStreamID)
	msg3 := core.NewMessage(nil, ([]byte)("reject"), 0, core.InvalidStreamID)

	result := filter.Modulate(msg1)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg2)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.Modulate(msg3)
	expect.Equal(core.ModulateResultDiscard, result)

	acceptExp := filter.exp
	filter.exp = nil

	result = filter.Modulate(msg1)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg2)
	expect.Equal(core.ModulateResultDiscard, result)

	result = filter.Modulate(msg3)
	expect.Equal(core.ModulateResultContinue, result)

	filter.expNot = nil

	result = filter.Modulate(msg1)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg2)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg3)
	expect.Equal(core.ModulateResultContinue, result)

	filter.exp = acceptExp

	result = filter.Modulate(msg1)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg2)
	expect.Equal(core.ModulateResultContinue, result)

	result = filter.Modulate(msg3)
	expect.Equal(core.ModulateResultDiscard, result)
}
