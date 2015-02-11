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
// - producer.Console
//	Formatter: "format.Runlength"
//	RunlengthDataFormatter: "format.Timestamp"
//
// RunlengthDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Runlength struct {
	base shared.Formatter
}

func init() {
	shared.RuntimeType.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPlugin(conf.GetString("RunlengthFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	return nil
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format Runlength) GetLength(msg shared.Message) int {
	msgLen := format.base.GetLength(msg)
	if msgLen < 10 {
		return 2 + msgLen
	}

	headerLen := int(math.Log10(float64(msgLen)) + 2)
	return headerLen + msgLen
}

// String returns the message as string
func (format Runlength) String(msg shared.Message) string {
	return fmt.Sprintf("%d:%s", format.base.GetLength(msg), format.base.String(msg))
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format Runlength) CopyTo(msg shared.Message, dest []byte) {
	len := copy(dest, strconv.Itoa(format.base.GetLength(msg)))
	dest[len] = ':'
	format.base.CopyTo(msg, dest[len+1:])
}
