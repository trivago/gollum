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

func TestAgent(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Agent")
	pluginConfig, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	plugin, casted := pluginConfig.(*Agent)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.0 Mobile/14E304 Safari/602.1"), nil, core.InvalidStreamID)
	err = plugin.ApplyFormatter(msg)
	expect.NoError(err)

	metadata := msg.GetMetadata()

	expect.MapEqual(metadata, "platform", "iPhone")
	expect.MapEqual(metadata, "os", "CPU iPhone OS 10_3_1 like Mac OS X")
	expect.MapEqual(metadata, "localization", "")
	expect.MapEqual(metadata, "browser", "Safari")
}
