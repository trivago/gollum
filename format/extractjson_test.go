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

func TestExtractJSON(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ExtractJSON")
	config.Override("Field", "test")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*ExtractJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"bar\",\"test\":\"valid\"}"),
		core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("valid", string(msg.Data()))
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
		core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("999999999", string(msg.Data()))
}
