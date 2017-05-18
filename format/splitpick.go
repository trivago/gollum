package format

import (
	"bytes"

	"github.com/trivago/gollum/core"
)

// SplitPick formatter plugin
// SplitPick separates value of messages according to a specified delimiter
// and returns the given indexed message. The index are zero based.
//
// Configuration example
//
//  - format.SplitPick:
//	  Index: 0
//	  Delimiter: ":"
//	  ApplyTo: "payload" # payload or <metaKey>
//
// SplitPickIndex defaults to 0.
//
// SplitPickDelimiter defaults to  ":".
//
// ApplyTo defines the formatter content to use
type SplitPick struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	index     int
	delimiter []byte
}

func init() {
	core.TypeRegistry.Register(SplitPick{})
}

// Configure initializes the SplitPick formatter plugin
func (format *SplitPick) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.index = conf.GetInt("Index", 0)
	format.delimiter = []byte(conf.GetString("Delimiter", ":"))

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *SplitPick) ApplyFormatter(msg *core.Message) error {
	parts := bytes.Split(format.GetAppliedContent(msg), format.delimiter)

	if format.index < len(parts) {
		format.SetAppliedContent(msg, parts[format.index])
	} else {
		format.SetAppliedContent(msg, []byte{})
	}

	return nil
}
