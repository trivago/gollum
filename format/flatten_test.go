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

	"github.com/trivago/tgo/tcontainer"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFlatten(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Flatten")
	config.Override("Source", "root")
	pluginConfig, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	plugin, casted := pluginConfig.(*Flatten)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"root": tcontainer.MarshalMap{
			"a": "test",
			"b": []string{"1", "2", "3"},
			"c": tcontainer.MarshalMap{
				"a": "test",
			},
		},
	}

	msg := core.NewMessage(nil, []byte("payload"), metadata, core.InvalidStreamID)
	err = plugin.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())

	metadata = msg.GetMetadata()

	expect.MapEqual(metadata, "root.a", "test")
	expect.MapEqual(metadata, "root.b", []interface{}{"1", "2", "3"})
	expect.MapEqual(metadata, "root.c.a", "test")
}
