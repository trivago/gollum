package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestSplitPick_Success(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitPick")
	config.Override("SplitPickIndex", 0)
	config.Override("SplitPickDelimiter", "#")
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
	config.Override("SplitPickIndex", 2)
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2:NjU0MzIx"), core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)

	expect.NoError(err)
	expect.Equal(0, len(msg.Data()))

}
