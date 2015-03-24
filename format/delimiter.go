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
	"github.com/trivago/gollum/shared"
	"io"
	"strings"
)

// Delimiter is a formatter that allows postfixing a message with a delimiter
// string.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Delimiter"
//     DelimiterDataFormatter: "format.Forward"
//     Delimiter: "\r\n"
//
// Delimiter defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
//
// DelimiterDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Delimiter struct {
	base      core.Formatter
	delimiter string
	msg       core.Message
	length    int
}

var delimiterEscapeChars = strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

func init() {
	shared.RuntimeType.Register(Delimiter{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Delimiter) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("DelimiterDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	format.delimiter = delimiterEscapeChars.Replace(conf.GetString("Delimiter", core.DefaultDelimiter))
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Delimiter) PrepareMessage(msg core.Message) {
	format.base.PrepareMessage(msg)
	format.length = format.base.Len() + len(format.delimiter)
}

// Len returns the length of a formatted message.
func (format *Delimiter) Len() int {
	return format.length
}

// String returns the message as string
func (format *Delimiter) String() string {
	return format.base.String() + format.delimiter
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Delimiter) Read(dest []byte) (int, error) {
	len, err := format.base.Read(dest)
	if err != nil {
		return len, err
	}
	len += copy(dest[len:], format.delimiter)
	return len, nil
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *Delimiter) WriteTo(writer io.Writer) (int64, error) {
	baseLen, err := format.base.WriteTo(writer)
	if err != nil {
		return baseLen, err
	}

	var len int
	len, err = writer.Write([]byte(format.delimiter))
	return baseLen + int64(len), err
}
