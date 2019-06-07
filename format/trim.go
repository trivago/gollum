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

// Trim formatter plugin
//
// Trim removes a set of characters from the beginning and end of a metadata value
// or the payload.
//
// Parameters
//
// - Characters: This value defines which characters should be removed from
// both ends of the data. The data to operate on is expected to be a string.
// By default this is set to " \t\r\n\v\f".
//
// Examples
//
// This example will trim spaces from the message payload:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Trim: {}
type Trim struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	characters           string `config:"Characters" default:" \t\r\n\v\f"`
}

func init() {
	core.TypeRegistry.Register(Trim{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Trim) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Trim) ApplyFormatter(msg *core.Message) error {
	content := format.GetSourceDataAsString(msg)
	format.SetTargetData(msg, strings.Trim(content, format.characters))
	return nil
}
