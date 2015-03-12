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

// Timestamp is a formatter that allows prefixing a message with a timestamp
// (time of arrival at gollum) as well as postfixing it with a delimiter string.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Timestamp"
//     TimestampDataFormatter: "format.Delimiter"
//     Timestamp: "2006-01-02T15:04:05.000 MST | "
//
// Timestamp defines a Go time format string that is used to format the actual
// timestamp that prefixes the message.
// By default this is set to "2006-01-02 15:04:05 MST | "
//
// TimestampDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Delimiter"
type Timestamp struct {
	base            shared.Formatter
	timestampFormat string
	timestamp       string
	length          int
}

func init() {
	shared.RuntimeType.Register(Timestamp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Timestamp) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPluginWithType(conf.GetString("TimestampDataFormatter", "format.Delimiter"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	format.timestampFormat = conf.GetString("Timestamp", shared.DefaultTimestamp)

	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Timestamp) PrepareMessage(msg shared.Message) {
	format.base.PrepareMessage(msg)
	format.length = format.base.Len() + len(format.timestampFormat)
	format.timestamp = msg.Timestamp.Format(format.timestampFormat)
}

// Len returns the length of a formatted message.
func (format *Timestamp) Len() int {
	return format.length
}

// String returns the message as string
func (format *Timestamp) String() string {
	return format.timestamp + format.base.String()
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Timestamp) Read(dest []byte) (int, error) {
	len := copy(dest, format.timestamp)
	baseLen, err := format.base.Read(dest[len:])
	return len + baseLen, err
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *Timestamp) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write([]byte(format.timestamp))
	if err != nil {
		return int64(len), err
	}

	var baseLen int64
	baseLen, err = format.base.WriteTo(writer)
	return int64(len) + baseLen, err
}
