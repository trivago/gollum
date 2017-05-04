package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"fmt"
)

func TestFormatterTrim(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Trim")
	config.Override("LeftSeparator", "|")
	config.Override("RightSeparator", "|")

	plugin, err := core.NewPlugin(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Trim)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("|foo bar foobar|"),
		0, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
	fmt.Println(msg.String())
}

func TestFormatterTrimWithSpaces(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Trim")
	config.Override("LeftSeparator", " ")
	config.Override("RightSeparator", "  ")

	plugin, err := core.NewPlugin(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Trim)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte(" foo bar foobar  "),
		0, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo bar foobar", msg.String())
	fmt.Println(msg.String())
}