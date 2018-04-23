// Copyright 2015-2018 trivago N.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package format

import (
	"strings"

	"github.com/trivago/gollum/core"
)

// MetadataCopy formatter plugin
//
// This formatter sets metadata fields by copying data from the message's
// payload or from other metadata fields.
//
// Parameters
//
// - Key: Defines the key to copy, i.e. the "source". ApplyTo will define
// the target of the copy, i.e. the "destination". An empty string will
// use the message payload as source.
// By default this parameter is set to an empty string (i.e. payload).
//
// - Mode: Defines the copy mode to use. This can be one of "append",
// "prepend" or "replace".
// By default this parameter is set to "replace".
//
// - Separator: When using mode prepend or append, defines the characters
// inserted between source and destination.
// By default this parameter is set to an empty string.
//
// - CopyToKeys: DEPRECATED. A list of meta data keys to copy the payload
// or metadata content to. If this field contains at least one value, mode
// is set to replace and the key field is ignored.
// By default this parameter is set to an empty list.
//
// Examples
//
// This example copies the payload to the field key and applies a hash on
// it contain a hash over the complete payload.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.MetadataCopy:
//        ApplyTo: key
//      - formatter.Identifier
//        Generator: hash
//        ApplyTo: key
//
type MetadataCopy struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	key                  string   `config:"Key"`
	separator            []byte   `config:"Separator"`
	metaDataKeys         []string `config:"CopyToKeys"` // deprecated
	mode                 metadataCopyMode
}

type metadataCopyMode int

const (
	metadataCopyModeAppend  = metadataCopyMode(iota)
	metadataCopyModeReplace = metadataCopyMode(iota)
	metadataCopyModePrepend = metadataCopyMode(iota)
)

func init() {
	core.TypeRegistry.Register(MetadataCopy{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *MetadataCopy) Configure(conf core.PluginConfigReader) {
	mode := conf.GetString("Mode", "replace")
	switch strings.ToLower(mode) {
	case "replace":
		format.mode = metadataCopyModeReplace
	case "append":
		format.mode = metadataCopyModeAppend
	case "prepend":
		format.mode = metadataCopyModePrepend
	default:
		conf.Errors.Pushf("mode must be one of replace, append or prepend")
	}
}

// ApplyFormatter update message payload
func (format *MetadataCopy) ApplyFormatter(msg *core.Message) error {

	if len(format.metaDataKeys) > 0 {
		// DEPRECATED
		// This codepath will be removed in 0.6
		meta := msg.GetMetadata()
		data := format.GetAppliedContent(msg)

		for _, key := range format.metaDataKeys {
			bufferCopy := make([]byte, len(data))
			copy(bufferCopy, data)
			meta.SetValue(key, bufferCopy)
		}
		return nil
	}

	getSourceData := core.GetAppliedContentGetFunction(format.key)
	srcData := getSourceData(msg)

	switch format.mode {
	case metadataCopyModeReplace:
		format.SetAppliedContent(msg, srcData)

	case metadataCopyModePrepend:
		dstData := format.GetAppliedContent(msg)
		if len(format.separator) != 0 {
			srcData = append(srcData, format.separator...)
		}
		format.SetAppliedContent(msg, append(srcData, dstData...))

	case metadataCopyModeAppend:
		dstData := format.GetAppliedContent(msg)
		if len(format.separator) != 0 {
			dstData = append(dstData, format.separator...)
		}
		format.SetAppliedContent(msg, append(dstData, srcData...))
	}

	return nil
}
