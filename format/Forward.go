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
}

func init() {
	shared.RuntimeType.Register(Forward{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Forward) Configure(conf shared.PluginConfig) error {
	return nil
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format Forward) GetLength(msg shared.Message) int {
	return len(msg.Data)
}

// String returns the message as string
func (format Forward) String(msg shared.Message) string {
	return msg.Data
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format Forward) CopyTo(dest []byte, msg shared.Message) {
	copy(dest, msg.Data)
}
