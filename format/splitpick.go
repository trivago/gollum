package format

import (
	"bytes"

	"github.com/trivago/gollum/core"
)

// SplitPick formatter plugin
// SplitPick separates value of messages according to a specified delimiter
// and returns the given indexed message. The index are zero based.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.SplitPick"
//	  SplitPickIndex: 0
//	  SplitPickDelimiter: ":"
//
//	By default, SplitPickIndex is 0.
//	By default, SplitPickDelimiter is ":".
type SplitPick struct {
	core.SimpleFormatter
	index     int
	delimiter []byte
}

func init() {
	core.TypeRegistry.Register(SplitPick{})
}

// Configure initializes the SplitPick formatter plugin
func (format *SplitPick) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.index = conf.GetInt("SplitPickIndex", 0)
	format.delimiter = []byte(conf.GetString("SplitPickDelimiter", ":"))

	return conf.Errors.OrNil()
}

// Modulate splits the message based on a delimiter and returns the indexed
// part.
// If the index is out of bound, an error is logged and the returned byte is empty.
func (format *SplitPick) Modulate(msg *core.Message) core.ModulateResult {
	parts := bytes.Split(msg.Data(), format.delimiter)

	if format.index < len(parts) {
		msg.Store(parts[format.index])
	} else {
		msg.Store([]byte{})
	}

	return core.ModulateResultContinue
}
