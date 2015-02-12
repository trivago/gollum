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
	base    shared.Formatter
	message *bytes.Buffer
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

// PrepareMessage sets the message to be formatted.
func (format *JSON) PrepareMessage(msg shared.Message) {
	format.base.PrepareMessage(msg)

	formattedMessage := format.base.String()
	format.message = bytes.NewBufferString("{\"message\":\"")

	json.HTMLEscape(format.message, []byte(formattedMessage))
	format.message.WriteString("\"}")
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *JSON) GetLength() int {
	return format.message.Len()
}

// String returns the message as string
func (format *JSON) String() string {
	return format.message.String()
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *JSON) CopyTo(dest []byte) int {
	return copy(dest, format.message.Bytes())
}
