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

func TestFormatterTrimToBounds(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", "|")
	config.Override("RightBounds", "|")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|foo bar foobar|"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
}

func TestFormatterTrimToBoundsOffset(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", "|")
	config.Override("RightBounds", "|")
	config.Override("LeftOffset", "1")
	config.Override("RightOffset", "1")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("||a||"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("a", msg.String())
}

func TestFormatterTrimToBoundsEmpty(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", "|")
	config.Override("RightBounds", "|")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("||"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("", msg.String())
}

func TestFormatterTrimToBoundsOverlap(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", "|")
	config.Override("RightBounds", "|")
	config.Override("LeftOffset", "1")
	config.Override("RightOffset", "1")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|a|"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("", msg.String())
}

func TestFormatterTrimToBoundsWithSpaces(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", " ")
	config.Override("RightBounds", "  ")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(" foo bar foobar  "), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
}

func TestFormatterTrimToBoundsTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.TrimToBounds")
	config.Override("LeftBounds", "|")
	config.Override("RightBounds", "|")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*TrimToBounds)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|foo bar foobar|"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("|foo bar foobar|second foo bar|"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("|foo bar foobar|", msg.String())
	expect.Equal("foo bar foobar|second foo bar", string(foo))
}
