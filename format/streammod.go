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
	"bytes"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// StreamMod is a formatter that modifies a message's stream by reading a prefix
// from the message's data (and discarding it).
// The prefix is defined by everything before a given delimiter in the
// message. If no delimiter is found or the prefix is empty the message stream
// is not changed.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.StreamMod"
//     StreamModFormatter: "format.Forward"
//     StreamModDelimiter: "$"
//
// StreamModFormatter defines the formatter applied after reading the stream.
// This formatter is applied to the data after StreamModDelimiter.
// By default this is set to "format.Forward"
//
// StreamModDelimiter defines the delimiter to search when extracting the stream
// name. By default this is set to ":".
type StreamMod struct {
	base      core.Formatter
	delimiter []byte
}

func init() {
	shared.RuntimeType.Register(StreamMod{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamMod) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("StreamModFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.delimiter = []byte(conf.GetString("StreamModDelimiter", ":"))
	format.base = plugin.(core.Formatter)

	return nil
}

// Format adds prefix and postfix to the message formatted by the base formatter
func (format *StreamMod) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	modMsg := msg
	prefixEnd := bytes.Index(msg.Data, format.delimiter)

	switch prefixEnd {
	case -1:
	case 0:
		modMsg.Data = msg.Data[1:]
	default:
		firstSpaceIdx := bytes.IndexByte(msg.Data, ' ')
		if (firstSpaceIdx < 0) || (firstSpaceIdx >= prefixEnd) {
			modMsg.StreamID = core.GetStreamID(string(msg.Data[:prefixEnd]))
			modMsg.Data = msg.Data[prefixEnd+1:]
		}
	}

	return format.base.Format(modMsg)
}
