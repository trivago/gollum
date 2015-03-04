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
	"fmt"
	"github.com/trivago/gollum/shared"
)

// Sequence is a formatter that allows prefixing a message with the message's
// sequence number
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Sequence"
//     SequenceDataFormatter: "format.Delimiter"
//
// SequenceDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Sequence struct {
	base         shared.Formatter
	length       int
	prefixLength int
	sequence     uint64
}

func init() {
	shared.RuntimeType.Register(Sequence{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Sequence) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPluginWithType(conf.GetString("SequenceDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Sequence) PrepareMessage(msg shared.Message) {
	format.base.PrepareMessage(msg)

	format.prefixLength = 2
	if msg.Sequence > 9 {
		format.prefixLength = shared.ItoLen(msg.Sequence) + 1
	}

	format.sequence = msg.Sequence
	format.length = format.base.GetLength() + int(format.prefixLength)
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Sequence) GetLength() int {
	return format.length
}

// String returns the message as string
func (format *Sequence) String() string {
	return fmt.Sprintf("%d:%s", format.sequence, format.base.String())
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Sequence) CopyTo(dest []byte) int {
	colonIdx := format.prefixLength - 1
	shared.Itobe(format.sequence, dest[:colonIdx])
	dest[colonIdx] = ':'

	return format.prefixLength + format.base.CopyTo(dest[format.prefixLength:])
}
