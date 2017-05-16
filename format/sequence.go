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
//  - format.Sequence:
//      Separator: ":"
//      ApplyTo: "payload" # payload or <metaKey>
//
// Separator sets the separator character placed after the sequence
// number. This is set to ":" by default. If no separator is set the sequence string will only set.
//
// ApplyTo defines the formatter content to use
type Sequence struct {
	core.SimpleFormatter
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
	separatorLen := len(format.separator)

	var payload []byte
	if separatorLen > 0 {
		content := format.GetAppliedContent(msg)

		dataSize := len(sequenceStr) + separatorLen + len(content)
		payload = core.MessageDataPool.Get(dataSize)

		offset := copy(payload, []byte(sequenceStr))
		offset += copy(payload[offset:], format.separator)
		copy(payload[offset:], content)
	} else {
		payload = []byte(sequenceStr)
	}

	format.SetAppliedContent(msg, payload)
	return nil
}
