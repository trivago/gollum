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

// ToJSON formatter
//
// This formatter converts metadata to JSON and stores it where applied.
//
// Examples
//
// This example removes the "pipe" key from the metadata produced by
// consumer.Console.
//
//  exampleProducer:
//    Type: consumer.Producer
//    Streams: stdin
//    Modulators:
//      - format.ToJSON
//        Ignore: ["foo"]
type ToJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	ignore               []string `config:"Ignore"`
}

func init() {
	core.TypeRegistry.Register(ToJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ToJSON) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *ToJSON) ApplyFormatter(msg *core.Message) error {
	metadata := msg.TryGetMetadata()
	if len(format.ignore) == 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		format.SetTargetData(msg, data)
		return nil
	}

	// TODO: This is rather expensive, but required as k can be a path.
	metadata = metadata.Clone()
	for _, k := range format.ignore {
		metadata.Delete(k)
	}

	format.SetTargetData(msg, nil)
	return nil
}
