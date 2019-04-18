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

func TestTemplate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Template")
	config.Override("Template", "{{ .foo }} {{ .test }}")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Template)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo":  "bar",
		"test": "valid",
	}

	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("bar valid", msg.String())
}

func TestTemplateSource(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Template")
	config.Override("Template", "{{ .foo }} {{ .test }}")
	config.Override("Source", "foo")
	config.Override("Target", "result")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Template)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": tcontainer.MarshalMap{
			"test": "valid",
			"foo":  "bar",
		},
	}

	msg := core.NewMessage(nil, []byte("payload"), metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	result, err := msg.GetMetadata().String("result")
	expect.NoError(err)
	expect.Equal("payload", msg.String())
	expect.Equal("bar valid", result)
}
