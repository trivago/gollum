package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFormatterRegExp(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.RegExp")
	config.Override("Expression", "([a-z]*)")
	config.Override("Template", "${1}")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*RegExp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test 123"), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", string(msg.Data()))
}

func TestFormatterRegExpApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.RegExp")
	config.Override("Expression", "([a-z]*)")
	config.Override("Template", "${1}")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*RegExp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("test 123"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("PAYLOAD", string(msg.Data()))
	expect.Equal("test", msg.MetaData().GetValueString("foo"))
}
