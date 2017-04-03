package format

import (
	"bytes"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// SplitPick formatter plugin
// SplitPick seperates value of messages according to a specified delimiter
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
	index     int
	delimiter []byte
	base      core.Formatter
}

func init() {
	shared.TypeRegistry.Register(SplitPick{})
}

// Configure initializes the SplitPick formatter plugin
func (format *SplitPick) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("SplitPickFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.base = plugin.(core.Formatter)

	format.index = conf.GetInt("SplitPickIndex", 0)
	format.delimiter = []byte(conf.GetString("SplitPickDelimiter", ":"))
	return nil
}

// Format splits the message based on a delimiter and returns the indexed
// part.
// If the index is out of bound, an error is logged and the returned byte is empty.
func (format *SplitPick) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	parts := bytes.Split(data, format.delimiter)

	if format.index < len(parts) {
		return parts[format.index], streamID
	}

	return []byte{}, streamID
}
