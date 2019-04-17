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

	msg := core.NewMessage(nil, []byte("test 123"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", string(msg.GetPayload()))
}

func TestFormatterRegExpTarget(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.RegExp")
	config.Override("Expression", "([a-z]*)")
	config.Override("Template", "${1}")
	config.Override("Target", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*RegExp)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("test 123"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	foo, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal("PAYLOAD", string(msg.GetPayload()))
	expect.Equal("test", string(foo))
}
