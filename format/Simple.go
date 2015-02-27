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
	"strings"
)

// Simple is a formatter that allows postfixing a message with a delimiter
// string.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Simple"
//     Delimiter: "\r\n"
//
// Delimiter defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
type Simple struct {
	delimiter    string
	msg          shared.Message
	delimiterLen int
	length       int
}

func init() {
	shared.RuntimeType.Register(Simple{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Simple) Configure(conf shared.PluginConfig) error {
	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")
	format.delimiter = escapeChars.Replace(conf.GetString("Delimiter", shared.DefaultDelimiter))
	format.delimiterLen = len(format.delimiter)
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Simple) PrepareMessage(msg shared.Message) {
	format.msg = msg
	format.length = len(format.msg.Data) + format.delimiterLen
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Simple) GetLength() int {
	return format.length
}

// String returns the message as string
func (format *Simple) String() string {
	return fmt.Sprintf("%s%s", string(format.msg.Data), format.delimiter)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Simple) CopyTo(dest []byte) int {
	len := copy(dest, format.msg.Data)
	len += copy(dest[len:], format.delimiter)
	return len
}
