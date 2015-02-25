package format

import (
	"github.com/trivago/gollum/shared"
)

// Forward is a formatter that passes a message as is
// Configuration example
//
// - producer.Console
//	Formatter: "format.Forward"
type Forward struct {
	msg shared.Message
}

func init() {
	shared.RuntimeType.Register(Forward{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Forward) Configure(conf shared.PluginConfig) error {
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Forward) PrepareMessage(msg shared.Message) {
	format.msg = msg
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Forward) GetLength() int {
	return len(format.msg.Data)
}

// String returns the message as string
func (format *Forward) String() string {
	return string(format.msg.Data)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Forward) CopyTo(dest []byte) int {
	return copy(dest, format.msg.Data)
}
