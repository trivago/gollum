package format

import (
	"testing"

	"fmt"
	"os"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestFormatterHostname(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(fmt.Sprintf("%s:%s", hostname, "test"), string(msg.GetPayload()))
}

func TestFormatterHostnameSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	config.Override("Separator", "-")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(fmt.Sprintf("%s-%s", hostname, "test"), string(msg.GetPayload()))
}

func TestFormatterHostnameNoSeparator(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	config.Override("Separator", "")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(fmt.Sprintf("%s%s", hostname, "test"), msg.String())
}

func TestFormatterHostnameApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	config.Override("ApplyTo", "foo")
	config.Override("Separator", "")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	val, err := msg.GetMetadata().Bytes("foo")
	expect.NoError(err)
	expect.Equal(hostname, string(val))
}
