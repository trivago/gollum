package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"os"
	"fmt"
)

func TestFormatterHostname(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)


	msg := core.NewMessage(nil, []byte("test"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(fmt.Sprintf("%s:%s", hostname, "test"), string(msg.Data()))
}

func TestFormatterHostnameSeperator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	config.Override("Separator", "-")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)


	msg := core.NewMessage(nil, []byte("test"), 0, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(fmt.Sprintf("%s-%s", hostname, "test"), string(msg.Data()))
}