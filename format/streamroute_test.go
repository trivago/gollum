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
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestStreamRoute(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")

	plugin, err := core.NewPlugin(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(core.LogInternalStream+":test"),
		0, core.DroppedStreamID)

	result := formatter.Modulate(msg)
	expect.Equal(core.ModulateResultRoute, result)

	expect.Equal("test", msg.String())
	expect.Equal(core.LogInternalStreamID, msg.StreamID())
}

// TODO
/*
func TestStreamRouteFormat(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.StreamRoute")
	config.Override("Formatter", map[string]string{"format.Envelope"})
	config.Override("EnvelopePrefix", "_")
	config.Override("EnvelopePostfix", "_")
	config.Override("StreamRouteFormatStream", true)

	plugin, err := core.NewPlugin(config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("GOLLUM:test"), 0)
	msg.StreamID = core.DroppedStreamID

	formatter.Format(msg)

	expect.Equal("test", string(msg.Data))
	expect.Equal(core.LogInternalStream, core.StreamRegistry.GetStreamName(msg.StreamID))
}*/
