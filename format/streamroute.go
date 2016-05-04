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
//    StreamRouteDelimiter: ":"
//    StreamRouteFormatStream: false
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
	core.SimpleFormatter
	streamFormatters []core.Formatter
	delimiter        []byte
}

func init() {
	core.TypeRegistry.Register(StreamRoute{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamRoute) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.delimiter = []byte(conf.GetString("Delimiter", ":"))
	plugins := conf.GetPluginArray("NameFormatters", []core.Plugin{})

	for _, plugin := range plugins {
		formatter, isFormatter := plugin.(core.Formatter)
		if !isFormatter {
			conf.Errors.Pushf("Plugin is not a valid formatter")
		} else {
			format.streamFormatters = append(format.streamFormatters, formatter)
		}
	}

	return conf.Errors.OrNil()
}

// Format adds prefix and postfix to the message formatted by the base formatter
func (format *StreamRoute) Format(msg *core.Message) {
	delimiterIdx := bytes.Index(msg.Data(), format.delimiter)

	switch delimiterIdx {
	case -1:
		return

	case 0:
		msg.Offset(len(format.delimiter))

	default:
		streamName := msg.Data()[:delimiterIdx]
		msg.Offset(delimiterIdx + len(format.delimiter))

		if len(format.streamFormatters) > 0 {
			nameMessage := core.NewMessage(nil, streamName, msg.Sequence(), msg.StreamID())
			for _, formatter := range format.streamFormatters {
				formatter.Format(nameMessage)
			}
			streamName = nameMessage.Data()
		}

		msg.SetStreamID(core.GetStreamID(string(streamName)))
	}
}
