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
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

type applyFormatterMockA struct {
	core.SimpleFormatter
}

// appends "A" to content
func (formatter *applyFormatterMockA) ApplyFormatter(msg *core.Message) error {
	c := formatter.GetAppliedContent(msg)
	new := []byte("A")
	c = append(c, new...)

	formatter.SetAppliedContent(msg, c)

	return nil
}

var configInjection string

type applyFormatterMockB struct {
	core.SimpleFormatter
}

// set "foo" to global var
func (formatter *applyFormatterMockB) Configure(conf core.PluginConfigReader) {
	configInjection = conf.GetString("foo", "")
}

// appends "B" to content
func (formatter *applyFormatterMockB) ApplyFormatter(msg *core.Message) error {
	c := formatter.GetAppliedContent(msg)
	new := []byte("B")
	c = append(c, new...)

	formatter.SetAppliedContent(msg, c)

	return nil
}

func TestAggregate_ApplyFormatter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	core.TypeRegistry.Register(applyFormatterMockA{})
	core.TypeRegistry.Register(applyFormatterMockB{})

	config := core.NewPluginConfig("", "format.Aggregate")
	config.Override("ApplyTo", "")

	config.Override("Modulators", []interface{}{
		"format.applyFormatterMockA",
		map[string]interface{}{
			"format.applyFormatterMockB": map[string]interface{}{
				"foo": "bar",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*Aggregate)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("fooAB", string(msg.GetPayload()))
	expect.Equal("bar", configInjection)
}

func TestAggregate_ApplyFormatterWithApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	core.TypeRegistry.Register(applyFormatterMockA{})
	core.TypeRegistry.Register(applyFormatterMockB{})

	config := core.NewPluginConfig("", "format.Aggregate")
	config.Override("ApplyTo", "foo")

	config.Override("Modulators", []interface{}{
		"format.applyFormatterMockA",
		map[string]interface{}{
			"format.applyFormatterMockB": map[string]interface{}{
				"ApplyTo": "bar",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)
	formatter, casted := plugin.(*Aggregate)
	expect.True(casted)

	metadata := core.Metadata{}
	metadata.SetValue("foo", []byte("value"))
	msg := core.NewMessage(nil, []byte("payload"), metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", string(msg.GetPayload()))
	expect.Equal("", configInjection)
	expect.Equal("valueAB", msg.GetMetadata().GetValueString("foo"))
}
