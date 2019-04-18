package format

import (
	"strings"

	"github.com/trivago/gollum/core"
)

// SplitToFields formatter
//
// This formatter splits data into an array by using the given delimiter and
// stores it at the metadata key denoted by Fields.
//
// Parameters
//
// - Delimiter: Defines the delimiter to use when splitting the data.
// By default this parameter is set to ","
//
// - Fields: Defines a index-to-key mapping for storing the resulting list into
// Metadata. If there are less entries in the resulting array than fields, the
// remaining fields will not be set. If there are more entries, the additional
// indexes will not be handled.
// By default this parameter is set to an empty list.
//
// Examples
//
// This example will split the payload by ":" and writes up to three elements
// as keys "first", "second" and "third" as fields below the field "values".
//
//  ExampleProducer:
//    Type: proucer.Console
//    Streams: console
//    Modulators:
//      - format.SplitToFields:
//        Target: values
//        Delimiter: ":"
//		  Fields: [first,second,third]
type SplitToFields struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	delimiter            string   `config:"Delimiter" default:","`
	fields               []string `config:"Fields"`
}

func init() {
	core.TypeRegistry.Register(SplitToFields{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *SplitToFields) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *SplitToFields) ApplyFormatter(msg *core.Message) error {
	parts := strings.Split(format.GetSourceDataAsString(msg), format.delimiter)
	tree := format.ForceTargetAsMetadata(msg)

	count := len(parts)
	if len(format.fields) < len(parts) {
		count = len(format.fields)
	}

	for i := 0; i < count; i++ {
		tree.Set(format.fields[i], parts[i])
	}

	return nil
}
