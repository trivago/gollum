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

func TestStreamRoute(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(core.LogInternalStream+":test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", msg.String())
	expect.Equal(core.LogInternalStreamID, msg.GetStreamID())
}

func TestStreamRouteNoStreamName(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(":test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", msg.String())
	expect.Equal(core.InvalidStreamID, msg.GetStreamID())
}

func TestStreamRouteFormat(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")
	config.Override("StreamModulator", []interface{}{
		map[interface{}]interface{}{
			"format.Envelope": map[string]string{
				"Prefix":  "_",
				"Postfix": "_",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("GOLLUM:test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", string(msg.GetPayload()))
	expect.Equal(core.LogInternalStream, core.StreamRegistry.GetStreamName(msg.GetStreamID()))
}

func TestStreamTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte(core.LogInternalStream+":test"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("payload", msg.String())
	expect.Equal("test", string(foo))
	expect.Equal(core.LogInternalStreamID, msg.GetStreamID())
}
