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
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestCopyReplace(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("not applied"), tcontainer.MarshalMap{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo", msg.String())
}

func TestCopyAddKey(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "")
	config.Override("Target", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("test", msg.String())
	expect.Equal("test", string(val))
}

func TestCopyAppend(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "foo")
	config.Override("Mode", "append")
	config.Override("Separator", " ")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), tcontainer.MarshalMap{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test foo", msg.String())
}

func TestCopyPrepend(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "foo")
	config.Override("Mode", "prepend")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), tcontainer.MarshalMap{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("footest", msg.String())
}

func TestCopyMetadataIntegrity(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "")
	config.Override("Target", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("payload", msg.String())
	expect.Equal("payload", string(foo))

	msg.StorePayload([]byte("xxx"))

	foo, err = msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("xxx", msg.String())
	expect.Equal("payload", string(foo))
}

func TestCopyPayloadIntegrity(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Copy")
	config.Override("Source", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Copy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("metadata"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("metadata", msg.String())
	expect.Equal("metadata", string(foo))

	msg.GetMetadata().Set("foo", []byte("xxx"))

	foo, err = msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("metadata", msg.String())
	expect.Equal("xxx", string(foo))
}
