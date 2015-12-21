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
	base     core.Formatter
	length   int
	sequence string
}

func init() {
	tgo.TypeRegistry.Register(Sequence{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Sequence) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("SequenceFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the sequence number of the message (followed by ":") to the
// message.
func (format *Sequence) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	basePayload, stream := format.base.Format(msg)
	baseLength := len(basePayload)
	sequenceStr := strconv.FormatUint(msg.Sequence, 10) + ":"

	payload := make([]byte, len(sequenceStr)+baseLength)
	len := copy(payload, []byte(sequenceStr))
	copy(payload[len:], basePayload)

	return payload, stream
}
