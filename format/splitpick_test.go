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
	plugin, err := core.NewPlugin(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2#NjU0MzIx"), 0, core.InvalidStreamID)
	result := formatter.Modulate(msg)

	expect.Equal(core.ModulateResultContinue, result)
	expect.Equal("MTIzNDU2", msg.String())

}

func TestSplitPick_OutOfBoundIndex(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.SplitPick")
	config.Override("SplitPickIndex", 2)
	plugin, err := core.NewPlugin(config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2:NjU0MzIx"), 0, core.InvalidStreamID)
	result := formatter.Modulate(msg)

	expect.Equal(core.ModulateResultContinue, result)

	expect.Equal(0, len(msg.Data()))

}
