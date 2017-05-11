package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"os"
	"strings"
)

func TestMetaDataCopy(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// mock config
	setupModulators, err := tcontainer.ConvertToMarshalMap(
		map[string]interface{}{"Modulators": []string{"format.Hostname"}},
		strings.ToLower)

	setupConf, err := tcontainer.ConvertToMarshalMap(
		map[string]interface{}{"foo": setupModulators},
		strings.ToLower)

	config := core.NewPluginConfig("", "format.MetaDataCopy")
	config.Override("WriteTo", []interface{}{
		setupConf,
		"bar",
	})
	// --

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetaDataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test payload"),
		0, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown host"
	}

	expect.Equal("test payload", msg.String())
	expect.True(strings.Contains(string(msg.MetaData().GetValue("foo", []byte{})), hostname))
	expect.Equal("test payload", string(msg.MetaData().GetValue("bar", []byte{})))
}

func TestMetaDataCopyApplyToHandling(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// mock config
	config := core.NewPluginConfig("", "format.MetaDataCopy")
	config.Override("WriteTo", []interface{}{
		"bar",
	})
	config.Override("ApplyTo", "meta:foo")
	// --

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetaDataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test payload"),
		0, core.InvalidStreamID)

	msg.MetaData().SetValue("foo", []byte("meta data string"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test payload", msg.String())
	expect.Equal("meta data string", string(msg.MetaData().GetValue("bar", []byte{})))
}
