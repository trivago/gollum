package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"strings"
)

func TestProcessJSONRename(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("Directives", []interface{}{
		"foo:rename:foobar",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1\",\"bar\":\"value2\"}"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.GetPayload())
	expect.True(strings.Contains(msgData, "\"foobar\":\"value1\""))
	expect.True(strings.Contains(msgData, "\"bar\":\"value2\""))
}

func TestProcessJSONReplace(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("Directives", []interface{}{
		"foo:replace:value:new",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1\",\"bar\":\"value2\"}"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.GetPayload())
	expect.True(strings.Contains(msgData, "\"foo\":\"new1\""))
}

func TestProcessJsonTrimValues(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("Directives", []interface{}{
		"foo:rename:foo2",
		"bar:rename:bar2",
	})
	config.Override("TrimValues", true)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1 \",\"bar\":\" value2\"}"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.GetPayload())

	expect.True(strings.Contains(msgData, "\"foo2\":\"value1\""))
	expect.True(strings.Contains(msgData, "\"bar2\":\"value2\""))
}

func TestProcessJsonTrimValuesFalse(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("Directives", []interface{}{
		"foo:rename:foo2",
		"bar:rename:bar2",
	})
	config.Override("TrimValues", false)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1 \",\"bar\":\" value2\"}"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.GetPayload())

	expect.True(strings.Contains(msgData, "\"foo2\":\"value1 \""))
	expect.True(strings.Contains(msgData, "\"bar2\":\" value2\""))
}

func TestProcessJSONApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("Directives", []interface{}{
		"test:rename:foo",
	})
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("TEST PAYLOAD"), nil, core.InvalidStreamID)
	msg.GetMetadata().SetValue("foo", []byte("{\"test\":\"foobar\"}"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.GetPayload())
	expect.Equal(msgData, "TEST PAYLOAD")
	expect.True(strings.Contains(msg.GetMetadata().GetValueString("foo"), "\"foo\":\"foobar\""))
}
