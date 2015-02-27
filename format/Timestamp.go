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

const (
	messageFormatTimestampSeparator = " | "
)

// Timestamp is a formatter that allows prefixing a message with a timestamp
// (time of arrival at gollum) as well as postfixing it with a delimiter string.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Timestamp"
//     Timestamp: "2006-01-02T15:04:05.000 MST"
//     Delimiter: "\r\n"
//
// Timestamp defines a Go time format string that is used to format the actual
// timestamp that prefixes the message.
// By default this is set to "2006-01-02 15:04:05 MST"
//
// Delimiter defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
type Timestamp struct {
	timestampFormat string
	delimiter       string
	msg             shared.Message
	timestamp       string
	formatLen       int
	length          int
}

func init() {
	shared.RuntimeType.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf shared.PluginConfig) error {
	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	format.delimiter = escapeChars.Replace(conf.GetString("Delimiter", shared.DefaultDelimiter))
	format.timestampFormat = conf.GetString("Timestamp", shared.DefaultTimestamp)
	format.formatLen = len(format.timestampFormat) + len(messageFormatTimestampSeparator) + len(format.delimiter)

	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Timestamp) PrepareMessage(msg shared.Message) {
	format.msg = msg
	format.length = len(format.msg.Data) + format.formatLen
	format.timestamp = format.msg.Timestamp.Format(format.timestampFormat)
}

// GetLength returns the length of a formatted message returned by String()
// or CopyTo().
func (format *Timestamp) GetLength() int {
	return format.length
}

// String returns the message as string
func (format *Timestamp) String() string {
	return fmt.Sprintf("%s%s%s%s", format.timestamp, messageFormatTimestampSeparator, string(format.msg.Data), format.delimiter)
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Timestamp) CopyTo(dest []byte) int {
	len := copy(dest[:], format.timestamp)
	len += copy(dest[len:], messageFormatTimestampSeparator)
	len += copy(dest[len:], format.msg.Data)
	len += copy(dest[len:], format.delimiter)
	return len
}
