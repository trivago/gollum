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

package format

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestBase64(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	pluginEncode, err := core.NewPluginWithType("format.Base64Encode", config)
	expect.NoError(err)
	pluginDecode, err := core.NewPluginWithType("format.Base64Decode", config)
	expect.NoError(err)

	encoder, castedEncoder := pluginEncode.(*Base64Encode)
	expect.True(castedEncoder)
	decoder, castedDecoder := pluginDecode.(*Base64Decode)
	expect.True(castedDecoder)

	msg := core.NewMessage(nil, []byte("test"), 0)
	msg.Data, _ = encoder.Format(msg)

	result, _ := decoder.Format(msg)

	expect.Equal("test", string(result))
}
