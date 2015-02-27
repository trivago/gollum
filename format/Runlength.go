package format

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"math"
	"strconv"
)

// Runlength is a formatter that prepends the length of the message, followed by
// a ":". The actual message is formatted by a nested formatter.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Runlength"
//     RunlengthDataFormatter: "format.Timestamp"
//
// RunlengthDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Runlength struct {
	base       shared.Formatter
	baseLength int
	length     int
}

func init() {
	shared.RuntimeType.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPlugin(conf.GetString("RunlengthDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Runlength) PrepareMessage(msg shared.Message) {
	format.base.PrepareMessage(msg)
	format.baseLength = format.base.GetLength()

	if format.baseLength < 10 {
		format.length = format.baseLength + 2
	} else {
		format.length = format.baseLength + int(math.Log10(float64(format.baseLength))+2)
	}
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Runlength) GetLength() int {
	return format.length
}

// String returns the message as string
func (format *Runlength) String() string {
	return fmt.Sprintf("%d:%s", format.baseLength, format.base.String())
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Runlength) CopyTo(dest []byte) int {
	len := copy(dest, strconv.Itoa(format.baseLength))
	dest[len] = ':'
	len++
	return len + format.base.CopyTo(dest[len:])
}
