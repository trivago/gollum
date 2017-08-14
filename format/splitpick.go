package format

import (
	"bytes"

	"github.com/trivago/gollum/core"
)

// SplitPick formatter
//
// This formatter splits data into an array by using the given delimiter and
// extracts the given index from that array. The value of that index will be
// written back.
//
// Parameters
//
// - Delimiter: Defines the delimiter to use when splitting the data.
// By default this parameter is set to ":"
//
// - Index: Defines the index to pick.
// By default this parameter is set to 0.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.SplitPick:
//        Index: 2
//        Delimiter: ":"
type SplitPick struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	index                int    `config:"Index" default:"0"`
	delimiter            []byte `config:"Delimiter" default:":"`
}

func init() {
	core.TypeRegistry.Register(SplitPick{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *SplitPick) Configure(conf core.PluginConfigReader) {
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
