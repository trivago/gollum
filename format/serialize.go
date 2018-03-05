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
	"github.com/trivago/gollum/core"
)

// Serialize formatter plugin
//
// Serialize is a formatter that serializes a message for later retrieval.
// The formatter uses the internal protobuf based function from msg.Serialize().
//
// Examples
//
// This example serializes all consumed messages:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Serialize
//
type Serialize struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Serialize{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Serialize) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Serialize) ApplyFormatter(msg *core.Message) error {
	data, err := msg.Serialize()
	if err != nil {
		format.Logger.Error(err)
		return err
	}

	format.SetAppliedContent(msg, data)
	return nil
}
