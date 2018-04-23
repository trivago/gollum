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

// Envelope formatter
//
// This formatter adds content to the beginning and/or end of a message.
//
// Parameters
//
// - Prefix: Defines a string that is added to the front of the message.
// Special characters like \n \r or \t can be used without additional escaping.
// By default this parameter is set to "".
//
// - Postfix: Defines a string that is added to the end of the message.
// Special characters like \n \r or \t can be used without additional escaping.
// By default this parameter is set to "\n".
//
// Examples
//
// This example adds a line number and a newline character to each message
// printed to the console.
//
//  exampleProducer:
//    Type: producer.Console
//    Streams: "*"
//    Modulators:
//      - format.Sequence
//      - format.Envelope
type Envelope struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	postfix              string `config:"Postfix" default:"\n"`
	prefix               string `config:"Prefix"`
}

func init() {
	core.TypeRegistry.Register(Envelope{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Envelope) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Envelope) ApplyFormatter(msg *core.Message) error {
	prefixLen := len(format.prefix)
	postfixLen := len(format.postfix)
	content := format.GetAppliedContent(msg)
	offset := 0

	payload := make([]byte, prefixLen+len(content)+postfixLen)

	if prefixLen > 0 {
		offset = copy(payload, format.prefix)
	}

	offset += copy(payload[offset:], content)

	if postfixLen > 0 {
		copy(payload[offset:], format.postfix)
	}

	format.SetAppliedContent(msg, payload)
	return nil
}
