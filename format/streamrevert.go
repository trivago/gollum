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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// StreamRevert formatter plugin
// StreamRevert is a formatter that recovers the last used stream from a message
// and sets it as a new target stream. Streams change whenever the Stream.Route
// or Message.Route function is used. This e.g. happens after a Drop call.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.StreamRevert"
//    StreamRevertFormatter: "format.Forward"
//
// StreamRevertFormatter defines the formatter applied after reading the stream.
// This formatter is applied to the data after StreamRevertDelimiter.
// By default this is set to "format.Forward"
type StreamRevert struct {
	base      core.Formatter
	delimiter []byte
}

func init() {
	shared.TypeRegistry.Register(StreamRevert{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *StreamRevert) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("StreamRevertFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	return nil
}

// Format adds prefix and postfix to the message formatted by the base formatter
func (format *StreamRevert) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, _ := format.base.Format(msg)
	return data, msg.PrevStreamID
}
