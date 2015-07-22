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
	"github.com/trivago/gollum/shared"
)

// StreamName is a formatter that prefixes a message with the StreamName.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.StreamName"
//     StreamNameFormatter: "format.Envelope"
//
// StreamNameDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Envelope"
type StreamName struct {
	base        core.Formatter
	usePrevious bool
}

func init() {
	shared.RuntimeType.Register(StreamName{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamName) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("StreamNameFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.usePrevious = conf.GetBool("PreviousStreamName", false)
	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the StreamName of the message to the message.
func (format *StreamName) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	var streamName string
	data, streamID := format.base.Format(msg)

	switch {
	case !format.usePrevious:
		streamName = core.StreamTypes.GetStreamName(streamID)

	case streamID != msg.StreamID:
		streamName = core.StreamTypes.GetStreamName(msg.StreamID)

	default:
		streamName = core.StreamTypes.GetStreamName(msg.PrevStreamID)
	}

	payload := make([]byte, len(streamName)+len(data)+1)
	streamNameLen := copy(payload, []byte(streamName))
	payload[streamNameLen] = ' '
	copy(payload[streamNameLen+1:], data)

	return payload, streamID
}
