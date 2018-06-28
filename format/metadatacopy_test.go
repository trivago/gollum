package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestMetadataCopyReplace(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("Key", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.Metadata{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("foo", msg.String())
}

func TestMetadataCopyAddKey(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test", msg.String())
	expect.Equal("test", msg.GetMetadata().GetValueString("foo"))
}

func TestMetadataCopyAppend(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("Key", "foo")
	config.Override("Mode", "append")
	config.Override("Separator", " ")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.Metadata{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("test foo", msg.String())
}

func TestMetadataCopyPrepend(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("Key", "foo")
	config.Override("Mode", "prepend")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("test"), core.Metadata{"foo": []byte("foo")}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("footest", msg.String())
}

func TestMetadataCopyDeprecated(t *testing.T) {
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

func TestMetadataCopyApplyToHandlingDeprecated(t *testing.T) {
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

func TestMetadataCopyMetadataIntegrity(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("ApplyTo", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("payload"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())
	expect.Equal("payload", msg.GetMetadata().GetValueString("foo"))

	msg.StorePayload([]byte("xxx"))

	expect.Equal("xxx", msg.String())
	expect.Equal("payload", msg.GetMetadata().GetValueString("foo"))
}

func TestMetadataCopyPayloadIntegrity(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.MetadataCopy")
	config.Override("Key", "foo")

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*MetadataCopy)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, nil, core.InvalidStreamID)
	msg.GetMetadata().SetValue("foo", []byte("metadata"))

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("metadata", msg.String())
	expect.Equal("metadata", msg.GetMetadata().GetValueString("foo"))

	msg.GetMetadata().SetValue("foo", []byte("xxx"))

	expect.Equal("metadata", msg.String())
	expect.Equal("xxx", msg.GetMetadata().GetValueString("foo"))
}
