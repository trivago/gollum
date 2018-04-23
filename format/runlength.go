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
	"strconv"

	"github.com/trivago/gollum/core"
)

// Runlength formatter plugin
//
// Runlength is a formatter that prepends the length of the message, followed by
// a ":". The actual message is formatted by a nested formatter.
//
// Parameters
//
// - Separator: This value is used as separator.
// By default this parameter is set to ":".
//
// - StoreRunlengthOnly: If this value is set to "true" only the runlength will
// stored. This option is useful to e.g. create metadata fields only containing
// the length of the payload. When set to "true" the Separator parameter will
// be ignored.
// By default this parameter is set to false.
//
// Examples
//
// This example will store the length of the payload in a separate metadata
// field.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.MetadataCopy:
//        CopyToKeys: ["length"]
//      - format.Runlength:
//        ApplyTo: length
//        StoreRunlengthOnly: true
//
type Runlength struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator" default:":"`
	storeRunlengthOnly   bool   `config:"StoreRunlengthOnly" default:"false"`
}

func init() {
	core.TypeRegistry.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Runlength) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)
	contentLen := len(content)
	lengthStr := strconv.Itoa(contentLen)

	var payload []byte
	if !format.storeRunlengthOnly {
		dataSize := len(lengthStr) + len(format.separator) + contentLen
		payload = make([]byte, dataSize)

		offset := copy(payload, []byte(lengthStr))
		offset += copy(payload[offset:], format.separator)
		copy(payload[offset:], content)
	} else {
		payload = []byte(lengthStr)
	}

	format.SetAppliedContent(msg, payload)
	return nil
}
