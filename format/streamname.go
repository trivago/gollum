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
//  - "stream.Broadcast":
//    Formatter: "format.StreamName"
//    StreamNameFormatter: "format.Envelope"
//    StreamNameSeparator: " "
//    StreamNameHistory: false
//
// StreamNameFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Envelope"
//
// StreamNameHistory can be set to true to not use the current but the previous
// stream name. This can be useful to e.g. get the name of the stream messages
// were sent to the fallback from. By default this is set to false.
//
// StreamNameSeparator sets the separator character placed after the stream name.
// This is set to " " by default.
type StreamName struct {
	core.SimpleFormatter `gollumdoc:embed_type`
	separator   []byte
	usePrevious bool
}

func init() {
	core.TypeRegistry.Register(StreamName{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamName) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.separator = []byte(conf.GetString("Separator", ":"))
	format.usePrevious = conf.GetBool("UseHistory", false)
	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *StreamName) ApplyFormatter(msg *core.Message) error {
	var streamName string

	switch {
	case !format.usePrevious:
		streamName = core.StreamRegistry.GetStreamName(msg.StreamID())
	default:
		streamName = core.StreamRegistry.GetStreamName(msg.PreviousStreamID())
	}

	dataSize := len(streamName) + len(format.separator) + msg.Len()
	payload := core.MessageDataPool.Get(dataSize)

	offset := copy(payload, []byte(streamName))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], msg.Data())

	msg.Store(payload)
	return nil
}
