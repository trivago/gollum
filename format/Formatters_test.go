package format

import (
	"github.com/trivago/gollum/shared"
	"testing"
)

func testFormatter(t *testing.T, formatter shared.Formatter) bool {
	expect := shared.NewExpect(t)

	msgString := "formatter" //\ttest\r\n123 456\n"
	msg := shared.NewMessage(msgString, []shared.MessageStreamID{}, 0)

	formatter.PrepareMessage(msg)
	buffer := make([]byte, formatter.GetLength())
	result := true

	result = expect.IntEq(formatter.GetLength(), len(formatter.String())) && result
	result = expect.IntEq(formatter.GetLength(), formatter.CopyTo(buffer)) && result
	result = expect.StringEq(formatter.String(), string(buffer)) && result

	return result
}

func TestFormatters(t *testing.T) {
	conf := shared.PluginConfig{}
	formatters := shared.RuntimeType.GetRegistered("format.")

	if len(formatters) == 0 {
		t.Error("No formatters defined")
	}

	for _, name := range formatters {
		plugin, err := shared.RuntimeType.NewPlugin(name, conf)
		if err != nil {
			t.Errorf("Failed to create formatter %s", name)
		} else {
			if !testFormatter(t, plugin.(shared.Formatter)) {
				t.Errorf("Formatter %s tests failed", name)
			}
		}
	}
}
