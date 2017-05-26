package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFormatterTrim(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Trim")
	config.Override("LeftSeparator", "|")
	config.Override("RightSeparator", "|")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Trim)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|foo bar foobar|"), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
}

func TestFormatterTrimWithSpaces(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Trim")
	config.Override("LeftSeparator", " ")
	config.Override("RightSeparator", "  ")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Trim)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(" foo bar foobar  "), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
}

func TestFormatterTrimApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Trim")
	config.Override("LeftSeparator", "|")
	config.Override("RightSeparator", "|")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Trim)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|foo bar foobar|"), core.InvalidStreamID)
	msg.GetMetadata().SetValue("foo", []byte("|foo bar foobar|second foo bar|"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("|foo bar foobar|", msg.String())
	expect.Equal("foo bar foobar|second foo bar", msg.GetMetadata().GetValueString("foo"))
}
