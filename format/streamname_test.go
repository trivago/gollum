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

func TestStreamName(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamName")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.LogInternalStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(core.LogInternalStream+":test", msg.String())
}

func TestStreamNameHistory(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamName")
	config.Override("UseHistory", true)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.LogInternalStreamID)
	msg.SetStreamID(core.LogInternalStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(core.LogInternalStream+":test", msg.String())
}

func TestStreamNameApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamName")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), core.LogInternalStreamID)
	msg.MetaData().SetValue("foo", []byte("test"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())
	expect.Equal(core.LogInternalStream+":test", msg.MetaData().GetValueString("foo"))
}

func TestStreamNameApplyToNoSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamName")
	config.Override("ApplyTo", "foo")
	config.Override("Separator", "")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), core.LogInternalStreamID)
	msg.MetaData().SetValue("foo", []byte("test"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())
	expect.Equal(core.LogInternalStream, msg.MetaData().GetValueString("foo"))
}
