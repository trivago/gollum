package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestSplitPick_Success(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitPick")
	config.Override("Index", 0)
	config.Override("Delimiter", "#")
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2#NjU0MzIx"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("MTIzNDU2", msg.String())
}

func TestSplitPick_OutOfBoundIndex(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitPick")
	config.Override("Index", 2)
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2:NjU0MzIx"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal(0, len(msg.Data()))
}

func TestSplitPickApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitPick")
	config.Override("Index", 0)
	config.Override("Delimiter", "#")
	config.Override("ApplyTo", "foo")
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("PAYLOAD"), core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte("MTIzNDU2#NjU0MzIx"))
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal("PAYLOAD", msg.String())
	expect.Equal("MTIzNDU2", msg.MetaData().GetValueString("foo"))
}