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

func TestExtractJSON(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ExtractJSON")
	config.Override("Field", "test")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ExtractJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"bar\",\"test\":\"valid\"}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("valid", string(msg.GetPayload()))
}

func TestExtractJSONPrecision(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ExtractJSON")
	config.Override("Field", "test")
	config.Override("Precision", 0)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ExtractJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"bar\",\"test\":999999999}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("999999999", string(msg.GetPayload()))
}

func TestExtractJSONApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ExtractJSON")
	config.Override("ApplyTo", "foo")
	config.Override("Field", "test")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ExtractJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"bar\"}"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("{\"foo\":\"bar\",\"test\":\"valid\"}"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("valid", string(val))
	expect.Equal("{\"foo\":\"bar\"}", msg.String())
}
