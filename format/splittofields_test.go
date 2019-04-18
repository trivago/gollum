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

func TestSplitToFieldsMoreValues(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitToFields")
	config.Override("Target", "values")
	config.Override("Fields", []string{"one", "two", "three"})
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitToFields)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("1,2,3,4"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	value, err := msg.GetMetadata().MarshalMap("values")
	expect.NoError(err)

	expect.MapEqual(value, "one", "1")
	expect.MapEqual(value, "two", "2")
	expect.MapEqual(value, "three", "3")
}

func TestSplitToFieldsLessValues(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitToFields")
	config.Override("Target", "values")
	config.Override("Fields", []string{"one", "two", "three"})
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitToFields)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("1,2"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	value, err := msg.GetMetadata().MarshalMap("values")
	expect.NoError(err)

	expect.MapEqual(value, "one", "1")
	expect.MapEqual(value, "two", "2")
	expect.MapNotSet(value, "three")
}
