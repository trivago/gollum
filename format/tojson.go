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
	"github.com/trivago/tgo/tcontainer"
)

// ToJSON formatter
//
// This formatter converts metadata to JSON and stores it where applied.
//
// Parameters
//
// - Root: The metadata key to transform to json. When left empty, all
// metadata is assumed. By default this is set to ''.
//
// - Ignore: A list of keys or paths to exclude from marshalling.
// please note that this is currently a quite expensive operation as
// all metadata below root is cloned during the process.
// By default this is set to an empty list.
//
// Examples
//
// This example transforms all metadata below the "foo" key to JSON and
// stores the result as the new payload.
//
//  exampleProducer:
//    Type: consumer.Producer
//    Streams: stdin
//    Modulators:
//      - format.ToJSON
//		  Root: "foo"
type ToJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	root                 string   `config:"Root"`
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
	if metadata == nil {
		format.SetTargetData(msg, []byte("{}"))
		return nil
	}

	if len(format.root) > 0 {
		val, exists := metadata.Value(format.root)
		if !exists {
			format.SetTargetData(msg, []byte("{}"))
			return nil
		}

		var err error
		metadata, err = tcontainer.ConvertToMarshalMap(val, nil)
		if err != nil {
			return err
		}
	}

	if len(format.ignore) > 0 {
		metadata = metadata.Clone()
		for _, k := range format.ignore {
			metadata.Delete(k)
		}
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	format.SetTargetData(msg, data)
	return nil
}
