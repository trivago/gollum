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
	"github.com/trivago/gollum/shared"
)

// Timestamp formatter plugin
// Timestamp is a formatter that allows prefixing a message with a timestamp
// (time of arrival at gollum) as well as postfixing it with a delimiter string.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Timestamp"
//    TimestampFormatter: "format.Envelope"
//    Timestamp: "2006-01-02T15:04:05.000 MST | "
//
// Timestamp defines a Go time format string that is used to format the actual
// timestamp that prefixes the message.
// By default this is set to "2006-01-02 15:04:05 MST | "
//
// TimestampFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Timestamp struct {
	base            core.Formatter
	timestampFormat string
}

func init() {
	shared.TypeRegistry.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("TimestampFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	format.timestampFormat = conf.GetString("Timestamp", "2006-01-02 15:04:05 MST | ")

	return nil
}

// Format prepends the timestamp of the message to the message.
func (format *Timestamp) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	basePayload, stream := format.base.Format(msg)
	baseLength := len(basePayload)
	timestampStr := msg.Timestamp.Format(format.timestampFormat)

	payload := make([]byte, len(timestampStr)+baseLength)
	len := copy(payload, []byte(timestampStr))
	copy(payload[len:], basePayload)

	return payload, stream
}
