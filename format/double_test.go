package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
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

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), nil, core.InvalidStreamID)
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

	msg := core.NewMessage(nil, []byte("TEST_VALUE"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE-TEST_VALUE", string(msg.GetPayload()))
}

func TestDoubleFormatterSource(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Double")

	config.Override("Source", "bar")
	config.Override("Left", []interface{}{
		tcontainer.MarshalMap{
			"format.Copy": tcontainer.MarshalMap{
				"Source": "foo",
				"Target": "bar",
			},
		},
	})
	config.Override("Right", []interface{}{
		tcontainer.MarshalMap{
			"format.Base64Encode": tcontainer.MarshalMap{
				"Source": "foo",
				"Target": "bar",
			},
		},
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Double)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("SOME_PAYLOAD_DATA"), nil, core.InvalidStreamID)
	msg.GetMetadata().Set("foo", []byte("TEST_VALUE"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("TEST_VALUE:VEVTVF9WQUxVRQ==", msg.String())
}
