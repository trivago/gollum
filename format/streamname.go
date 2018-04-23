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

// StreamName formatter
//
// This formatter prefixes data with the name of the current or previous stream.
//
// Parameters
//
// - UsePrevious: Set to true to use the name of the previous stream.
// By default this parameter is set to false.
//
// - Separator: Defines the separator string used between stream name and data.
// By default this parameter is set to ":".
//
// Examples
//
// This example prefixes the message with the most recent routing history.
//
//  exampleProducer:
//    Type: producer.Console
//    Streams: "*"
//    Modulators:
//      - format.StreamName:
//        Separator: ", "
//        UsePrevious: true
//      - format.StreamName:
//        Separator: ": "
type StreamName struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator" default:":"`
	usePrevious          bool   `config:"UsePrevious"`
}

func init() {
	core.TypeRegistry.Register(StreamName{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamName) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *StreamName) ApplyFormatter(msg *core.Message) error {
	streamName := format.getStreamName(msg)
	content := format.GetAppliedContent(msg)

	dataSize := len(streamName) + len(format.separator) + len(content)
	payload := make([]byte, dataSize)

	offset := copy(payload, []byte(streamName))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], content)

	format.SetAppliedContent(msg, payload)
	return nil
}

func (format *StreamName) getStreamName(msg *core.Message) string {
	var streamName string

	switch {
	case !format.usePrevious:
		streamName = core.StreamRegistry.GetStreamName(msg.GetStreamID())
	default:
		streamName = core.StreamRegistry.GetStreamName(msg.GetPrevStreamID())
	}

	return streamName
}
