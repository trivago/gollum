package format

import (
	"bytes"
	"encoding/json"
	"github.com/trivago/gollum/shared"
)

// JSON is a formatter that passes a message encapsulated as JSON in the form
// {"message":"..."}. The actual message is formatted by a nested formatter and
// HTML escaped.
// Configuration example
//
// - producer.Console
//	Formatter: "format.JSON"
//	JSONDataFormatter: "format.Timestamp"
//
// JSONDataFormatter defines the formatter for the data transferred as message.
// By default this is set to "format.Forward"
type JSON struct {
	base shared.Formatter
}

func init() {
	shared.RuntimeType.Register(JSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSON) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPlugin(conf.GetString("JSONDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	return nil
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format JSON) GetLength(msg shared.Message) int {
	return len(format.String(msg))
}

// String returns the message as string
func (format JSON) String(msg shared.Message) string {
	formattedMessage := format.base.String(msg)
	encodedMessage := bytes.NewBufferString("{\"message\":\"")

	json.HTMLEscape(encodedMessage, []byte(formattedMessage))
	encodedMessage.WriteString("\"}")

	return encodedMessage.String()
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format JSON) CopyTo(dest []byte, msg shared.Message) {
	formattedMessage := format.base.String(msg)
	encodedMessage := new(bytes.Buffer)

	encodedMessage.WriteString("{\"message\":\"")
	json.HTMLEscape(encodedMessage, []byte(formattedMessage))
	encodedMessage.WriteString("\"}")

	copy(dest, encodedMessage.Bytes())
}
