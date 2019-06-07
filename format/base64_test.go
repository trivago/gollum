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

func TestBase64(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Base64Encode")
	pluginEncode, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	config = core.NewPluginConfig("", "format.Base64Decode")
	pluginDecode, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	encoder, castedEncoder := pluginEncode.(*Base64Encode)
	expect.True(castedEncoder)
	decoder, castedDecoder := pluginDecode.(*Base64Decode)
	expect.True(castedDecoder)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = encoder.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal("dGVzdA==", string(msg.GetPayload()))

	err = decoder.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", string(msg.GetPayload()))
}

func TestBase64DecodeApplyHandling(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Base64Decode")
	config.Override("Source", "foo")
	config.Override("Target", "foo")
	pluginDecode, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	decoder, castedDecoder := pluginDecode.(*Base64Decode)
	expect.True(castedDecoder)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("dGVzdA=="))

	err = decoder.ApplyFormatter(msg)
	expect.NoError(err)

	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("test", string(val))
}

func TestBase64EncodeApplyHandling(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Base64Encode")
	config.Override("Source", "foo")
	config.Override("Target", "foo")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	encoder, casted := plugin.(*Base64Encode)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("test"))

	err = encoder.ApplyFormatter(msg)
	expect.NoError(err)

	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("dGVzdA==", string(val))
}
