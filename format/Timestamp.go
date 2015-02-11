package format

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"strings"
)

const (
	messageFormatTimestampSeparator = " | "
)

// Timestamp is a formatter that allows prefixing a message with a timestamp
// (time of arrival at gollum) as well as postfixing it with a delimiter string.
// Configuration example
//
// - producer.Console
//	Formatter: "format.Timestamp"
//	Timestamp: "2006-01-02T15:04:05.000 MST"
//	Delimiter: "\r\n"
//
// Timestamp defines a Go time format string that is used to format the actual
// timestamp that prefixes the message.
// By default this is set to "2006-01-02 15:04:05 MST"
//
// Delimiter defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
type Timestamp struct {
	timestampFormat string
	delimiter       string
}

func init() {
	shared.RuntimeType.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf shared.PluginConfig) error {
	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	format.delimiter = escapeChars.Replace(conf.GetString("Delimiter", shared.DefaultDelimiter))
	format.timestampFormat = conf.GetString("Timestamp", shared.DefaultTimestamp)

	return nil
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format Timestamp) GetLength(msg shared.Message) int {
	return len(format.timestampFormat) + len(messageFormatTimestampSeparator) + len(msg.Data) + len(format.delimiter)
}

// String returns the message as string
func (format Timestamp) String(msg shared.Message) string {
	return fmt.Sprintf("%s%s%s%s", msg.Timestamp.Format(format.timestampFormat), messageFormatTimestampSeparator, msg.Data, format.delimiter)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format Timestamp) CopyTo(msg shared.Message, dest []byte) {
	len := copy(dest[:], msg.Timestamp.Format(format.timestampFormat))
	len += copy(dest[len:], messageFormatTimestampSeparator)
	len += copy(dest[len:], msg.Data)
	len += copy(dest[len:], format.delimiter)
}
