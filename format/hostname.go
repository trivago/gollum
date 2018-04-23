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
	"os"

	"github.com/trivago/gollum/core"
)

// Hostname formatter
//
// This formatter prefixes the message or metadata with the hostname of
// the machine gollum is running on.
//
// Parameters
//
// - Separator: Defines the separator string placed between hostname and data.
// By default this parameter is set to ":".
//
// Examples
//
// This example inserts the hostname into an existing JSON payload.
//
//  exampleProducer:
//    Type: producer.Console
//    Streams: "*"
//    Modulators:
//      - format.Trim:
//        LeftSeparator: "{"
//        RightSeparator: "}"
//      - format.Hostname
//        Separator: ","
//      - format.Envelope:
//        Prefix: "{\"host\":"
//        Postfix: "}"
type Hostname struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator" default:":"`
}

func init() {
	core.TypeRegistry.Register(Hostname{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Hostname) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Hostname) ApplyFormatter(msg *core.Message) error {
	content := format.getFinalContent(format.GetAppliedContent(msg))
	format.SetAppliedContent(msg, content)

	return nil
}

func (format *Hostname) getFinalContent(content []byte) []byte {
	hostname, err := os.Hostname()
	if err != nil {
		format.Logger.Error(err)
		hostname = "unknown host"
	}

	dataSize := len(hostname) + len(format.separator) + len(content)
	payload := make([]byte, dataSize)

	offset := copy(payload, []byte(hostname))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], content)

	return payload
}
