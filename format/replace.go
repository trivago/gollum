package format

import (
	"strings"

	"github.com/trivago/gollum/core"
)

// Replace formatter
//
// This formatter replaces all occurrences in a string with another.
//
// Parameters
//
// - Search: Defines the string to search for. When left empty, the target will
// be completely replaced by ReplaceWith.
// By default this is set to "".
//
// - ReplaceWith: Defines the string to replace all occurences of "search" with.
// By default this is set to "".
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.Replace:
//        Search: "foo"
//        ReplaceWith: "bar"
type Replace struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	search               string `config:"Search" default:""`
	replaceWith          string `config:"ReplaceWith" default:""`
}

func init() {
	core.TypeRegistry.Register(Replace{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Replace) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Replace) ApplyFormatter(msg *core.Message) error {
	if len(format.search) == 0 {
		format.SetTargetData(msg, format.replaceWith)
		return nil
	}

	srcData := format.GetSourceDataAsString(msg)
	format.SetTargetData(msg, strings.Replace(srcData, format.search, format.replaceWith, -1))
	return nil
}
