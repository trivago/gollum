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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestStreamRoute(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")

	plugin, err := core.NewPluginWithType("format.StreamRoute", config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(core.LogInternalStream+":test"), 0)
	msg.StreamID = core.DroppedStreamID

	result, streamID := formatter.Format(msg)

	expect.Equal("test", string(result))
	expect.Equal(core.LogInternalStreamID, streamID)
}

func TestStreamRouteFormat(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("StreamRouteStreamFormatter", "format.Envelope")
	config.Override("EnvelopePrefix", "_")
	config.Override("EnvelopePostfix", "_")
	config.Override("StreamRouteFormatStream", true)

	plugin, err := core.NewPluginWithType("format.StreamRoute", config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamRoute)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("GOLLUM:test"), 0)
	msg.StreamID = core.DroppedStreamID

	result, streamID := formatter.Format(msg)

	expect.Equal("test", string(result))
	expect.Equal(core.LogInternalStream, core.StreamRegistry.GetStreamName(streamID))
}
