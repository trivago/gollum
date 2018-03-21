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
	"bytes"

	"github.com/trivago/gollum/core"
)

// StreamRoute formatter plugin
//
// StreamRoute is a formatter that modifies a message's stream by reading a
// prefix from the message's data (and discarding it).
// The prefix is defined as everything before a given delimiter in the
// message. If no delimiter is found or the prefix is empty the message stream
// is not changed.
//
// Parameters
//
// - Delimiter: This value defines the delimiter to search when extracting the stream name.
// By default this parameter is set to ":".
//
// - StreamModulator: A list of further modulators to format and filter the extracted stream name.
// By default this parameter is "empty".
//
// Examples
//
// This example sets the stream name for messages like `<error>:a message string` to `error`
// and `a message string` as payload:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.StreamRoute:
//        Delimiter: ":"
//        StreamModulator:
//          - format.Trim:
//            LeftSeparator: <
//            RightSeparator: >
type StreamRoute struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	streamModulators     core.ModulatorArray `config:"StreamModulator"`
	delimiter            []byte              `config:"Delimiter" default:":"`
}

func init() {
	core.TypeRegistry.Register(StreamRoute{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamRoute) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *StreamRoute) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)
	delimiterIdx := bytes.Index(content, format.delimiter)

	switch delimiterIdx {
	case 0:
		content = content[len(format.delimiter):]
		format.SetAppliedContent(msg, content)
	case -1:
		// no prefix

	default:
		streamName := content[:delimiterIdx]
		streamMsg := core.NewMessage(nil, streamName, nil, msg.GetStreamID())

		content = content[(delimiterIdx + len(format.delimiter)):]
		format.SetAppliedContent(msg, content)

		switch result := format.streamModulators.Modulate(streamMsg); result {
		case core.ModulateResultDiscard, core.ModulateResultFallback:
			return nil // ### return, rule based early out ###
		}

		targetStreamID := core.GetStreamID(streamMsg.String())
		if msg.GetStreamID() != targetStreamID {
			msg.SetStreamID(targetStreamID)
		}
	}

	return nil
}
