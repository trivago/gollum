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

// StreamRevert formatter
//
// This formatter gets the previously used stream from a message and sets it as
// the new target stream.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.StreamRevert
type StreamRevert struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(StreamRevert{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamRevert) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *StreamRevert) ApplyFormatter(msg *core.Message) error {
	msg.SetStreamID(msg.GetPrevStreamID())
	return nil
}
