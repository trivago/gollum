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
	message []byte
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
	format.message = msg.Data
}

// Len returns the length of a formatted message.
func (format *Forward) Len() int {
	return len(format.message)
}

// String returns the message as string
func (format *Forward) String() string {
	return string(format.message)
}

// Read copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Forward) Read(dest []byte) (int, error) {
	return copy(dest, format.message), nil
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *Forward) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write(format.message)
	return int64(len), err
}
