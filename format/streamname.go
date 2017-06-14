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
	"github.com/trivago/gollum/core"
)

// StreamName formatter plugin
// StreamName is a formatter that prefixes a message with the StreamName.
// Configuration example
//
//  - format.StreamName:
//      Separator: " "
//      UseHistory: false
//      ApplyTo: "payload" # payload or <metaKey>
//
// UseHistory can be set to true to not use the current but the previous
// stream name. This can be useful to e.g. get the name of the stream messages
// were sent to the fallback from. By default this is set to false.
//
// Separator sets the separator character placed after the stream name.
// This is set to " " by default.
//
// ApplyTo defines the formatter content to use
type StreamName struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator" default:":"`
	usePrevious          bool   `config:"UseHistory"`
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
	payload := core.MessageDataPool.Get(dataSize)

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
