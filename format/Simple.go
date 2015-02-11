package format

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"strings"
)

// Simple is a formatter that allows postfixing a message with a delimiter
// string.
// Configuration example
//
// - producer.Console
//	Formatter: "format.Simple"
//	Delimiter: "\r\n"
//
// Delimiter defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
type Simple struct {
	delimiter string
}

func init() {
	shared.RuntimeType.Register(Simple{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Simple) Configure(conf shared.PluginConfig) error {
	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")
	format.delimiter = escapeChars.Replace(conf.GetString("Delimiter", shared.DefaultDelimiter))
	return nil
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format Simple) GetLength(msg shared.Message) int {
	return len(msg.Data) + len(format.delimiter)
}

// String returns the message as string
func (format Simple) String(msg shared.Message) string {
	return fmt.Sprintf("%s%s", msg.Data, format.delimiter)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format Simple) CopyTo(msg shared.Message, dest []byte) {
	len := copy(dest, format.delimiter)
	len += copy(dest[len:], msg.Data)
}
