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

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE:VEVTVF9WQUxVRQ==", string(msg.GetPayload()))
}

func TestDoubleFormatterSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Double")
	config.Override("Separator", "-")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Double)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE-TEST_VALUE", string(msg.GetPayload()))
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

	msg := core.NewMessage(nil, []byte("SOME_PAYLOAD_DATA"), core.InvalidStreamID)
	msg.GetMetadata().SetValue("foo", []byte("TEST_VALUE"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE:VEVTVF9WQUxVRQ==", msg.GetMetadata().GetValueString("foo"))
	expect.Equal("SOME_PAYLOAD_DATA", msg.String())
}
