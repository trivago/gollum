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

func TestEnvelope(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Envelope")
	config.Override("Prefix", "start ")
	config.Override("Postfix", " end")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Envelope)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("start test end", string(msg.GetPayload()))
}

func TestEnvelopeTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Envelope")
	config.Override("Source", "foo")
	config.Override("Target", "foo")
	config.Override("Prefix", "start ")
	config.Override("Postfix", " end")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Envelope)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("bar"))
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("test", msg.String())
	expect.Equal("start bar end", string(val))
}
