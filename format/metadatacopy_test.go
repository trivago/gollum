package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestMetadataCopy(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("CopyToKeys", []string{"foo", "bar"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", msg.String())
	expect.Equal("test", msg.GetMetadata().GetValueString("foo"))
	expect.Equal("test", msg.GetMetadata().GetValueString("bar"))
}

func TestMetadataCopyApplyToHandling(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("ApplyTo", "foo")
	config.Override("CopyToKeys", []string{"bar"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), nil, core.InvalidStreamID)
	msg.GetMetadata().SetValue("foo", []byte("meta"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())
	expect.Equal("meta", msg.GetMetadata().GetValueString("foo"))
	expect.Equal("meta", msg.GetMetadata().GetValueString("bar"))
}
