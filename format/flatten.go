package format

import (
	"fmt"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// Flatten formatter
//
// This formatter takes a metadata tree and moves all subkeys on the same level
// as the root of the tree. Fields will be named according to their hierarchy
// but joining all keys in the path with a given separator.
//
// Parameters
//
// - Separator: Defines the separator used when joining keys.
// By default this parameter is set to "."
//
// Examples
//
// This will flatten all elements below the key "tree" on the root level.
// A key `/tree/a/b` will become `/tree.a.b`
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.Flatten:
//        Source: tree
type Flatten struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            string `config:"Separator" default:"."`
	prefix               string
}

func init() {
	core.TypeRegistry.Register(Flatten{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Flatten) Configure(conf core.PluginConfigReader) {
	format.prefix = conf.GetString("Source", "")

	if len(format.prefix) > 0 {
		format.prefix += format.separator
	}
}

func (format *Flatten) flatten(prefix string, root tcontainer.MarshalMap, target tcontainer.MarshalMap) {
	for k, v := range root {
		prefixedKey := prefix + k

		switch node := v.(type) {
		case tcontainer.MarshalMap:
			format.flatten(prefixedKey+format.separator, node, target)

		case map[string]interface{}:
			nodeAsMap, err := tcontainer.ConvertToMarshalMap(node, func(s string) string { return s })
			if err != nil {
				target.Set(prefixedKey, node)
			}
			format.flatten(prefixedKey+format.separator, nodeAsMap, target)

		default:
			target.Set(prefixedKey, node)
		}
	}
}

// ApplyFormatter update message payload
func (format *Flatten) ApplyFormatter(msg *core.Message) error {
	if !format.SourceIsMetadata() {
		return fmt.Errorf("flatten source must be a metadata key")
	}

	node, err := format.GetSourceAsMetadata(msg)
	if err != nil {
		return err
	}

	target := format.ForceTargetAsMetadata(msg)
	format.flatten(format.prefix, node, target)

	return nil
}
