package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFormatterIdentifier(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Identifier")
	config.Override("Use", "hash")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Identifier)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("f9e6e6ef197c2b25", string(msg.Data()))
}

func TestFormatterIdentifierApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Identifier")
	config.Override("ApplyTo", "foo")
	config.Override("Use", "hash")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Identifier)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("test"))
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("f9e6e6ef197c2b25", msg.MetaData().GetValueString("foo"))
	expect.Equal("payload", msg.String())
}
