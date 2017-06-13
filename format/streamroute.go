// Copyright 2015-2017 trivago GmbH
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
// StreamRoute is a formatter that modifies a message's stream by reading a
// prefix from the message's data (and discarding it).
// The prefix is defined by everything before a given delimiter in the
// message. If no delimiter is found or the prefix is empty the message stream
// is not changed.
// Configuration example
//
//  - format.StreamRoute:
//    StreamModulator:
//      - format.Forward
//    Delimiter: ":"
//    ApplyTo: "payload" # payload or <metaKey>
//
// StreamModulator is used when StreamRouteFormatStream is set to true.
// By default this is empty.
//
// Delimiter defines the delimiter to search when extracting the stream
// name. By default this is set to ":".
//
// StreamRouteFormatStream can be set to true to apply StreamRouteFormatter to both
// parts of the message (stream and data). Set to false by default.
//
// ApplyTo defines the formatter content to use
type StreamRoute struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	streamModulators     core.ModulatorArray `config:"StreamModulator"`
	delimiter            []byte              `config:"Delimiter" default:":"`
}

func init() {
	core.TypeRegistry.Register(StreamRoute{})
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
		streamMsg := core.NewMessage(nil, []byte(streamName), msg.GetStreamID())

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
