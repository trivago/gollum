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

package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestSequence(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Sequence")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Sequence)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), 10, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("10:test", string(msg.Data()))
}

func TestSequenceApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Sequence")
	config.Override("ApplyTo", "foo")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Sequence)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), 10, core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("test"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("PAYLOAD", string(msg.Data()))
	expect.Equal("10:test", msg.MetaData().GetValueString("foo"))
}

func TestSequenceApplyToNoSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Sequence")
	config.Override("ApplyTo", "foo")
	config.Override("Separator", "")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Sequence)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), 10, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("PAYLOAD", string(msg.Data()))
	expect.Equal("10", msg.MetaData().GetValueString("foo"))
}