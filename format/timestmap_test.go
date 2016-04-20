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
	"time"
)

func TestTimestamp(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	plugin, err := core.NewPluginWithType("format.Timestamp", config)
	expect.NoError(err)

	formatter, casted := plugin.(*Timestamp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), 0)
	msg.Timestamp = msg.Timestamp.Add(time.Hour + time.Minute + time.Second)
	prefix := msg.Timestamp.Format(formatter.timestampFormat)

	result, _ := formatter.Format(msg)
	expect.Equal(prefix+"test", string(result))

	msg.Timestamp = msg.Timestamp.Add(time.Hour + time.Minute + time.Second)

	result, _ = formatter.Format(msg)
	expect.Neq(prefix+"test", string(result))
}
