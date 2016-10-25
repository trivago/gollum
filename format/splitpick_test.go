package format

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestSplitPick_Success(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Settings["SplitPickIndex"] = 0
	config.Settings["SplitDelimiter"] = "#"
	plugin, err := core.NewPluginWithType("format.SplitPick", config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2#NjU0MzIx"), 0)
	result, _ := formatter.Format(msg)

	expect.Equal("MTIzNDU2", string(result))

}

func TestSplitPick_OutOfBoundIndex(t *testing.T) {
	expect := shared.NewExpect(t)

	config := core.NewPluginConfig("")
	config.Settings["SplitPickIndex"] = 2
	plugin, err := core.NewPluginWithType("format.SplitPick", config)

	expect.NoError(err)

	formatter, casted := plugin.(*SplitPick)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("MTIzNDU2:NjU0MzIx"), 0)
	result, _ := formatter.Format(msg)

	expect.Equal(0, len(result))

}
