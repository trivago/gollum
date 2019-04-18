package format

import (
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestSplit(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Split")
	config.Override("Target", "values")
	plugin, err := core.NewPluginWithConfig(config)

	expect.NoError(err)

	formatter, casted := plugin.(*Split)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("1,2,3"), nil, core.InvalidStreamID)
	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	value, err := msg.GetMetadata().StringArray("values")

	expect.NoError(err)
	expect.Equal([]string{"1", "2", "3"}, value)
}
