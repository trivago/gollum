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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"io"
	"strconv"
)

// Runlength is a formatter that prepends the length of the message, followed by
// a ":". The actual message is formatted by a nested formatter.
// Configuration example
//
//   - producer.Console
//     Formatter: "format.Runlength"
//     RunlengthDataFormatter: "format.Delimiter"
//
// RunlengthDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Runlength struct {
	base      core.Formatter
	length    int
	lengthStr string
}

func init() {
	shared.RuntimeType.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("RunlengthDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Runlength) PrepareMessage(msg core.Message) {
	format.base.PrepareMessage(msg)
	baseLength := format.base.Len()

	format.lengthStr = strconv.Itoa(baseLength) + ":"
	format.length = len(format.lengthStr) + baseLength
}

// Len returns the length of a formatted message.
func (format *Runlength) Len() int {
	return format.length
}

// String returns the message as string
func (format *Runlength) String() string {
	return fmt.Sprintf("%s%s", format.lengthStr, format.base.String())
}

// CopyTo copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Runlength) Read(dest []byte) (int, error) {
	len := copy(dest, []byte(format.lengthStr))
	baseLen, err := format.base.Read(dest[len:])
	return len + baseLen, err
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *Runlength) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write([]byte(format.lengthStr))
	if err != nil {
		return int64(len), err
	}

	var baseLen int64
	baseLen, err = format.base.WriteTo(writer)
	return int64(len) + baseLen, err
}
