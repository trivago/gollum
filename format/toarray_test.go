package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestToArray(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ToArray")
	config.Override("Keys", []interface{}{
		"foo",
		"bar",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ToArray)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": "value1",
		"bar": "value2",
	}
	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("value1,value2", string(msg.GetPayload()))
}

func TestToArrayApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ToArray")
	config.Override("ApplyTo", "baz")
	config.Override("Keys", []string{
		"foo",
		"bar",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ToArray)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": "value1",
		"bar": "value2",
	}
	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal([]byte{}, msg.GetPayload())

	val, err := msg.GetMetadata().String("baz")
	expect.NoError(err)
	expect.Equal("value1,value2", val)
}
