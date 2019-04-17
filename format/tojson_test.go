package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestToJSON(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ToJSON")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ToJSON)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": "value1",
		"root": tcontainer.MarshalMap{
			"a": "a",
			"b": 5,
			"c": []int{1, 2, 3},
		},
	}
	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(`{"foo":"value1","root":{"a":"a","b":5,"c":[1,2,3]}}`, string(msg.GetPayload()))
}

func TestToJSONWithIgnore(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ToJSON")
	config.Override("Ignore", []string{"root"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ToJSON)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": "value1",
		"root": tcontainer.MarshalMap{
			"a": "a",
			"b": 5,
			"c": []int{1, 2, 3},
		},
	}
	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(`{"foo":"value1"}`, string(msg.GetPayload()))
}

func TestToJSONWithRoot(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ToJSON")
	config.Override("Root", "root")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ToJSON)
	expect.True(casted)

	metadata := tcontainer.MarshalMap{
		"foo": "value1",
		"root": tcontainer.MarshalMap{
			"a": "a",
			"b": 5,
			"c": []int{1, 2, 3},
		},
	}
	msg := core.NewMessage(nil, []byte{}, metadata, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(`{"a":"a","b":5,"c":[1,2,3]}`, string(msg.GetPayload()))
}
