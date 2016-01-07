// Copyright 2015 trivago GmbH
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

// StreamName is a formatter that prefixes a message with the StreamName.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.StreamName"
//     StreamNameFormatter: "format.Envelope"
//
// StreamNameFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Envelope"
//
// StreamNameHistory can be set to true to not use the current but the previous
// stream name. This can be usefull to e.g. get the name of the stream messages
// were dropped from. By default this is set to false.
//
// StreamNameSeparator sets the separator character placed after the stream name.
// This is set to " " by default.
type StreamName struct {
	base        core.Formatter
	separator   string
	usePrevious bool
}

func init() {
	core.TypeRegistry.Register(StreamName{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamName) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("StreamNameFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.separator = conf.GetString("StreamNameSeparator", " ")
	format.usePrevious = conf.GetBool("StreamNameHistory", false)
	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the StreamName of the message to the message.
func (format *StreamName) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	var streamName string
	data, streamID := format.base.Format(msg)

	switch {
	case !format.usePrevious:
		streamName = core.StreamRegistry.GetStreamName(streamID)

	case streamID != msg.StreamID:
		streamName = core.StreamRegistry.GetStreamName(msg.StreamID)

	default:
		streamName = core.StreamRegistry.GetStreamName(msg.PrevStreamID)
	}

	payload := make([]byte, len(streamName)+len(format.separator)+len(data))
	streamNameLen := copy(payload, []byte(streamName))
	separatorLen := copy(payload[streamNameLen:], format.separator)
	copy(payload[streamNameLen+separatorLen:], data)

	return payload, streamID
}
