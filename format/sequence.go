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
	"strconv"
)

// Sequence is a formatter that allows prefixing a message with the message's
// sequence number
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Sequence"
//     SequenceFormatter: "format.Envelope"
//
// SequenceDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Sequence struct {
	core.FormatterBase
	separator string
}

func init() {
	core.TypeRegistry.Register(Sequence{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Sequence) Configure(conf core.PluginConfig) error {
	err := format.FormatterBase.Configure(conf)
	if err != nil {
		return err
	}
	format.separator = conf.GetString("Separator", ":")
	return nil
}

// Format prepends the sequence number of the message (followed by ":") to the
// message.
func (format *Sequence) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	sequenceStr := strconv.FormatUint(msg.Sequence, 10)

	payload := make([]byte, len(sequenceStr)+len(format.separator)+len(msg.Data))
	len := copy(payload, []byte(sequenceStr))
	len += copy(payload[len:], []byte(format.separator))

	copy(payload[len:], msg.Data)
	return payload, msg.StreamID
}
