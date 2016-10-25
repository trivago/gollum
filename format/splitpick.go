package format

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strings"
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
	delimiter string
	base      core.Formatter
}

func init() {
	shared.TypeRegistry.Register(SplitPick{})
}

// Configure initializes the SplitPick formatter plugin
func (format *SplitPick) Configure(conf core.PluginConfig) error {
	format.index = conf.GetInt("SplitPickIndex", 0)
	format.delimiter = conf.GetString("SplitDelimiter", ":")
	return nil
}

// Format splits the message based on a delimiter and returns the indexed
// part.
// If the index is out of bound, an error is logged and the returned byte is empty.
func (format *SplitPick) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data := msg.Data
	parts := strings.Split(string(data), format.delimiter)

	if len(parts) <= format.index {
		Log.Error.Println("Out of bound index for SplitPick formatter.")
		return nil, msg.StreamID
	}

	partOfInterest := parts[format.index]
	return []byte(partOfInterest), msg.StreamID
}
