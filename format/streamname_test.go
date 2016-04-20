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

func TestStreamName(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")

	plugin, err := core.NewPluginWithType("format.StreamName", config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), 0)
	msg.StreamID = core.DroppedStreamID

	result, _ := formatter.Format(msg)

	expect.Equal(core.DroppedStream+" test", string(result))
}

func TestStreamNameHistory(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Override("StreamNameHistory", true)

	plugin, err := core.NewPluginWithType("format.StreamName", config)
	expect.NoError(err)

	formatter, casted := plugin.(*StreamName)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), 0)
	msg.StreamID = core.DroppedStreamID
	msg.PrevStreamID = core.LogInternalStreamID

	result, _ := formatter.Format(msg)

	expect.Equal(core.LogInternalStream+" test", string(result))
}
