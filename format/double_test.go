package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestDoubleFormatter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Double")
	//config.Override("Left", []interface{}{
	//	"format.Base64Encode",
	//})

	config.Override("Right", []interface{}{
		"format.Base64Encode",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Double)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE:VEVTVF9WQUxVRQ==", string(msg.Data()))
}

func TestDoubleFormatterSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Double")
	config.Override("Separator", "-")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Double)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE-TEST_VALUE", string(msg.Data()))
}

func TestDoubleFormatterApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Double")
	config.Override("ApplyTo", "foo")

	config.Override("Right", []interface{}{
		"format.Base64Encode",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Double)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("SOME_PAYLOAD_DATA"), 0, core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("TEST_VALUE"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE:VEVTVF9WQUxVRQ==", msg.MetaData().GetValueString("foo"))
	expect.Equal("SOME_PAYLOAD_DATA", msg.String())
}