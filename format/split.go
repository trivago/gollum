package format

import (
	"fmt"
	"strings"

	"github.com/trivago/gollum/core"
)

// Split formatter
//
// This formatter splits data into an array by using the given delimiter and
// stores it at the metadata key denoted by target. Targeting the payload (by
// not given a target or passing an empty string) will result in an error.
//
// Parameters
//
// - Delimiter: Defines the delimiter to use when splitting the data.
// By default this parameter is set to ","
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.Split:
//        Target: values
//        Delimiter: ":"
type Split struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	delimiter            string `config:"Delimiter" default:","`
}

func init() {
	core.TypeRegistry.Register(Split{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Split) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Split) ApplyFormatter(msg *core.Message) error {
	if !format.TargetIsMetadata() {
		return fmt.Errorf("split target must be a metadata key")
	}

	parts := strings.Split(format.GetSourceDataAsString(msg), format.delimiter)
	format.SetTargetData(msg, parts)

	return nil
}
