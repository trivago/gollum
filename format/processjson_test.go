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
	config.Override("ProcessJSONDirectives", []interface{}{
		"foo:rename:foobar",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1\",\"bar\":\"value2\"}"), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.Data())
	expect.True(strings.Contains(msgData, "\"foobar\":\"value1\""))
	expect.True(strings.Contains(msgData, "\"bar\":\"value2\""))
}

func TestProcessJSONReplace(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ProcessJSON")
	config.Override("ProcessJSONDirectives", []interface{}{
		"foo:replace:value:new",
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ProcessJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"foo\":\"value1\",\"bar\":\"value2\"}"), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	msgData := string(msg.Data())
	expect.True(strings.Contains(msgData, "\"foo\":\"new1\""))
}
