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

package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestRunlength(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Runlength")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Runlength)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("4:test", string(msg.GetPayload()))
}

func TestRunlengthTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Runlength")
	config.Override("ApplyTo", "foo")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Runlength)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("test"))
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("PAYLOAD", string(msg.GetPayload()))
	expect.Equal("4:test", string(foo))
}

func TestRunlengthTargetAndStoreRunlengthOnly(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Runlength")
	config.Override("ApplyTo", "foo")
	config.Override("StoreRunlengthOnly", true)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Runlength)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("test"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("PAYLOAD", string(msg.GetPayload()))
	expect.Equal("4", string(foo))
}
