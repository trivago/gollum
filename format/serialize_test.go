package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFormatterSerialize(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Serialize")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Serialize)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo bar"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	controlMsg, err := core.DeserializeMessage(msg.GetPayload())
	expect.NoError(err)

	expect.Equal("foo bar", string(controlMsg.GetPayload()))
}

func TestFormatterSerializeApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Serialize")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Serialize)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("foo bar"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	controlMsg, err := core.DeserializeMessage(msg.GetMetadata().GetValue("foo"))
	expect.NoError(err)

	expect.Equal("foo bar", controlMsg.String())
	expect.Equal("foo bar", msg.String())
}
