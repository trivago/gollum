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

type applyFormatterMockA struct {
	core.SimpleFormatter
}

// appends "A" to content
func (formatter *applyFormatterMockA) ApplyFormatter(msg *core.Message) error {
	c := formatter.GetSourceDataAsString(msg)
	formatter.SetTargetData(msg, c+"A")

	return nil
}

type applyFormatterMockB struct {
	core.SimpleFormatter
	value string `config:"Value"`
}

// set "foo" to global var
func (formatter *applyFormatterMockB) Configure(conf core.PluginConfigReader) {
}

// appends "B" to content
func (formatter *applyFormatterMockB) ApplyFormatter(msg *core.Message) error {
	c := formatter.GetSourceDataAsString(msg)
	formatter.SetTargetData(msg, c+formatter.value)

	return nil
}

func TestAggregate_ApplyFormatter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	core.TypeRegistry.Register(applyFormatterMockA{})
	core.TypeRegistry.Register(applyFormatterMockB{})

	config := core.NewPluginConfig("", "format.Aggregate")

	config.Override("Modulators", []interface{}{
		map[string]interface{}{
			"format.applyFormatterMockA": map[string]interface{}{},
		},
		map[string]interface{}{
			"format.applyFormatterMockB": map[string]interface{}{
				"Value": "B",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*Aggregate)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("fooAB", string(msg.GetPayload()))
}

func TestAggregate_ApplyFormatterWithTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	core.TypeRegistry.Register(applyFormatterMockA{})
	core.TypeRegistry.Register(applyFormatterMockB{})

	config := core.NewPluginConfig("", "format.Aggregate")

	config.Override("Modulators", []interface{}{
		map[string]interface{}{
			"format.applyFormatterMockA": map[string]interface{}{},
		},
		map[string]interface{}{
			"format.applyFormatterMockB": map[string]interface{}{
				"Target": "foo",
				"Value":  "B",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*Aggregate)
	expect.True(casted)

	metadata := core.NewMetadata()
	metadata.Set("foo", "metadata")

	msg := core.NewMessage(nil, []byte("payload"), metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payloadA", string(msg.GetPayload()))

	val, err := msg.GetMetadata().String("foo")
	expect.NoError(err)
	expect.Equal("payloadAB", val)
}
