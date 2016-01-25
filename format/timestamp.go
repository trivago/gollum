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
	"github.com/trivago/tgo"
)

// Timestamp is a formatter that allows prefixing a message with a timestamp
// (time of arrival at gollum) as well as postfixing it with a delimiter string.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Timestamp"
//     TimestampFormatter: "format.Envelope"
//     Timestamp: "2006-01-02T15:04:05.000 MST | "
//
// Timestamp defines a Go time format string that is used to format the actual
// timestamp that prefixes the message.
// By default this is set to "2006-01-02 15:04:05 MST | "
//
// TimestampDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Timestamp struct {
	core.FormatterBase
	timestampFormat string
}

func init() {
	core.TypeRegistry.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf core.PluginConfig) error {
	errors := tgo.NewErrorStack()
	errors.Push(format.FormatterBase.Configure(conf))

	format.timestampFormat = errors.Str(conf.GetString("Timestamp", "2006-01-02 15:04:05 MST | "))
	return errors.ErrorOrNil()
}

// Format prepends the timestamp of the message to the message.
func (format *Timestamp) Format(msg core.Message) ([]byte, core.MessageStreamID) {

	timestampStr := msg.Timestamp.Format(format.timestampFormat)

	payload := make([]byte, len(timestampStr)+len(msg.Data))
	len := copy(payload, []byte(timestampStr))
	copy(payload[len:], msg.Data)

	return payload, msg.StreamID
}
