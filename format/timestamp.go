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

// Timestamp formatter plugin
//
// Timestamp is a formatter that allows prefixing messages with a timestamp
// (time of arrival at gollum). The timestamp format is freely configurable
// and can e.g. contain a delimiter sequence at the end.
//
// Parameters
//
// - Timestamp: This value defines a Go time format string that is used to f
// ormat the timestamp.
// By default this parameter is set to  "2006-01-02 15:04:05 MST | ".
//
// Examples
//
// This example will set a time string to the meta data field `time`:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Timestamp:
//        Timestamp: "2006-01-02T15:04:05.000 MST"
//        ApplyTo: time
type Timestamp struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	timestampFormat      string `config:"Timestamp" default:"2006-01-02 15:04:05 MST | "`
}

func init() {
	core.TypeRegistry.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Timestamp) ApplyFormatter(msg *core.Message) error {
	timestampStr := msg.GetCreationTime().Format(format.timestampFormat)
	content := format.GetAppliedContent(msg)

	dataSize := len(timestampStr) + len(content)
	payload := make([]byte, dataSize)

	offset := copy(payload, []byte(timestampStr))
	copy(payload[offset:], content)

	format.SetAppliedContent(msg, payload)
	return nil
}
