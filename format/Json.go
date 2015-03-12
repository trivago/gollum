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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/trivago/gollum/shared"
	"io"
)

// JSON is a formatter that passes a message encapsulated as JSON in the form
// {"message":"..."}. The actual message is formatted by a nested formatter and
// HTML escaped.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.JSON"
//     JSONDataFormatter: "format.Delimiter"
//
// JSONDataFormatter defines the formatter for the data transferred as message.
// By default this is set to "format.Forward"
type JSON struct {
	base    shared.Formatter
	message *bytes.Buffer
}

func init() {
	shared.RuntimeType.Register(JSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSON) Configure(conf shared.PluginConfig) error {
	plugin, err := shared.RuntimeType.NewPluginWithType(conf.GetString("JSONDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(shared.Formatter)
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *JSON) PrepareMessage(msg shared.Message) {
	format.base.PrepareMessage(msg)
	format.message = bytes.NewBufferString(fmt.Sprintf("{\"time\":\"%s\",\"seq\":%d,\"message\":\"", msg.Timestamp.Format(shared.DefaultTimestamp), msg.Sequence))

	json.HTMLEscape(format.message, []byte(format.base.String()))
	format.message.WriteString("\"}")
}

// Len returns the length of a formatted message.
func (format *JSON) Len() int {
	return format.message.Len()
}

// String returns the message as string
func (format *JSON) String() string {
	return format.message.String()
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *JSON) Read(dest []byte) (int, error) {
	return copy(dest, format.message.Bytes()), nil
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *JSON) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write(format.message.Bytes())
	return int64(len), err
}
