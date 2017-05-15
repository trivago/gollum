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
	"strconv"
)

// Sequence formatter plugin
// Sequence is a formatter that allows prefixing a message with the message's
// sequence number
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Sequence"
//    SequenceFormatter: "format.Envelope"
//    SequenceSeparator: ":"
//
// SequenceSeparator sets the separator character placed after the sequence
// number. This is set to ":" by default.
//
// SequenceDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Sequence struct {
	core.SimpleFormatter `gollumdoc:embed_type`
	separator []byte
}

func init() {
	core.TypeRegistry.Register(Sequence{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Sequence) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.separator = []byte(conf.GetString("Separator", ":"))
	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *Sequence) ApplyFormatter(msg *core.Message) error {
	sequenceStr := strconv.FormatUint(msg.Sequence(), 10)

	dataSize := len(sequenceStr) + len(format.separator) + msg.Len()
	payload := core.MessageDataPool.Get(dataSize)

	offset := copy(payload, []byte(sequenceStr))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], msg.Data())

	msg.Store(payload)
	return nil
}
