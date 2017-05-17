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

func TestClearFormatter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Clear")
	pluginConfig, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	plugin, casted := pluginConfig.(*Clear)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
	err = plugin.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("", msg.String())
}

func TestClearFormatterApplyHandling(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Clear")
	config.Override("ApplyTo", "foo")
	pluginConfig, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	plugin, casted := pluginConfig.(*Clear)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("bar"))

	err = plugin.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("", string(msg.MetaData().GetValueString("foo")))
	expect.Equal("test", msg.String())
}
