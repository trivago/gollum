package format

import (
	"testing"

	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"os"
)

func TestFormatterHostname(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Hostname")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Hostname)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
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

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
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

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)
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

	msg := core.NewMessage(nil, []byte("test"), core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, _ := os.Hostname()
	expect.Equal(hostname, msg.GetMetadata().GetValueString("foo"))
}
