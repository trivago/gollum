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
	"encoding/json"

	"github.com/trivago/gollum/core"
)

// JSON formatter
//
// This formatter parses json data into metadata.
//
// Examples
//
// This example parses the payload as JSON and stores it below the key
// "data".
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stdin
//    Modulators:
//      - format.JSON
//        Target: data
type JSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(JSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSON) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *JSON) ApplyFormatter(msg *core.Message) error {
	srcData := format.GetSourceDataAsBytes(msg)
	metadata := format.ForceTargetAsMetadata(msg)

	return json.Unmarshal(srcData, &metadata)
}
