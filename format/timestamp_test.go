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

func TestTimestamp(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Timestamp")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Timestamp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	prefix := msg.GetCreationTime().Format(formatter.timestampFormat)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(prefix+"test", msg.String())
}

func TestTimestampTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Timestamp")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Timestamp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	timestamp := msg.GetCreationTime().Format(formatter.timestampFormat)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("test", msg.String())
	expect.Equal(timestamp, string(foo))
}
