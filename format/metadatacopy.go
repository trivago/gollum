// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/gollum/core"
)

// MetadataCopy formatter plugin
//
// This formatter sets metadata fields by copying data from the message's
// payload or from other metadata fields.
//
// Parameters
//
// - CopyToKeys: A list of meta data keys to copy the payload or metadata
// content to.
// By default this parameter is set to an empty list.
//
// Examples
//
// This example copies the payload to the fields prefix and key. The prefix
// field will extract everything up to the first space as hostname, the key
// field will contain a hash over the complete payload.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.MetadataCopy:
//        CopyToKeys: ["prefix", "key"]
//      - format.SplitPick:
//        ApplyTo: prefix
//        Delimiter: " "
//        Index: 0
//      - formatter.Identifier
//        Generator: hash
//        ApplyTo: key
//
type MetadataCopy struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	metaDataKeys         []string `config:"CopyToKeys"`
}

func init() {
	core.TypeRegistry.Register(MetadataCopy{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *MetadataCopy) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *MetadataCopy) ApplyFormatter(msg *core.Message) error {
	meta := msg.GetMetadata()
	data := format.GetAppliedContent(msg)
	for _, key := range format.metaDataKeys {
		meta.SetValue(key, data)
	}
	return nil
}
