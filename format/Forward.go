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
	"github.com/trivago/gollum/shared"
	"io"
)

// Forward is a formatter that passes a message as is
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Forward"
type Forward struct {
	msg shared.Message
}

func init() {
	shared.RuntimeType.Register(Forward{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Forward) Configure(conf shared.PluginConfig) error {
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Forward) PrepareMessage(msg shared.Message) {
	format.msg = msg
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Forward) GetLength() int {
	return len(format.msg.Data)
}

// String returns the message as string
func (format *Forward) String() string {
	return string(format.msg.Data)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Forward) CopyTo(dest []byte) int {
	return copy(dest, format.msg.Data)
}

// Write writes the message to the given io.Writer.
func (format *Forward) Write(writer io.Writer) {
	writer.Write(format.msg.Data)
}
