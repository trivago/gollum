// Copyright 2015-2016 trivago GmbH
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
	"github.com/trivago/gollum/shared"
)

// StreamRoute formatter plugin
// StreamRoute is a formatter that modifies a message's stream by reading a
// prefix from the message's data (and discarding it).
// The prefix is defined by everything before a given delimiter in the
// message. If no delimiter is found or the prefix is empty the message stream
// is not changed.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.StreamRoute"
//    StreamRouteFormatter: "format.Forward"
//    StreamRouteStreamFormatter: "format.Forward"
//    StreamRouteDelimiter: "$"
//    StreamRouteFormatBoth: false
//
// StreamRouteFormatter defines the formatter applied after reading the stream.
// This formatter is applied to the data after StreamRouteDelimiter.
// By default this is set to "format.Forward"
//
// StreamRouteStreamFormatter is used when StreamRouteFormatStream is set to true.
// By default this is the same value as StreamRouteFormatter.
//
// StreamRouteDelimiter defines the delimiter to search when extracting the stream
// name. By default this is set to ":".
//
// StreamRouteFormatStream can be set to true to apply StreamRouteFormatter to both
// parts of the message (stream and data). Set to false by default.
type StreamRoute struct {
	base         core.Formatter
	streamFormat core.Formatter
	delimiter    []byte
	formatBoth   bool
}

func init() {
	shared.TypeRegistry.Register(StreamRoute{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamRoute) Configure(conf core.PluginConfig) error {
	baseFormatter := conf.GetString("StreamRouteFormatter", "format.Forward")
	plugin, err := core.NewPluginWithType(baseFormatter, conf)
	if err != nil {
		return err
	}

	streamPlugin, err := core.NewPluginWithType(conf.GetString("StreamRouteStreamFormatter", baseFormatter), conf)
	if err != nil {
		return err
	}

	format.delimiter = []byte(conf.GetString("StreamRouteDelimiter", ":"))
	format.base = plugin.(core.Formatter)
	format.streamFormat = streamPlugin.(core.Formatter)
	format.formatBoth = conf.GetBool("StreamRouteFormatStream", false)

	return nil
}

// Format adds prefix and postfix to the message formatted by the base formatter
func (format *StreamRoute) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	modMsg := msg
	prefixEnd := bytes.Index(msg.Data, format.delimiter)

	switch prefixEnd {
	case -1:
	case 0:
		modMsg.Data = msg.Data[1:]
	default:
		firstSpaceIdx := bytes.IndexByte(msg.Data, ' ')
		if (firstSpaceIdx < 0) || (firstSpaceIdx >= prefixEnd) {
			streamName := msg.Data[:prefixEnd]

			if format.formatBoth {
				streamMessage := core.NewMessage(nil, streamName, 0)
				streamName, _ = format.streamFormat.Format(streamMessage)
			}

			modMsg.StreamID = core.StreamRegistry.GetStreamID(string(streamName))
			modMsg.Data = msg.Data[prefixEnd+1:]
		}
	}

	return format.base.Format(modMsg)
}
