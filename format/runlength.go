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

// Runlength formatter plugin
// Runlength is a formatter that prepends the length of the message, followed by
// a ":". The actual message is formatted by a nested formatter.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Runlength"
//    RunlengthSeparator: ":"
//    RunlengthFormatter: "format.Envelope"
//
// RunlengthSeparator sets the separator character placed after the runlength.
// This is set to ":" by default.
//
// RunlengthDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Runlength struct {
	core.SimpleFormatter
	separator []byte
}

func init() {
	core.TypeRegistry.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.separator = []byte(conf.GetString("Separator", ":"))
	return conf.Errors.OrNil()
}

// Modulate prepends the length of the message (followed by ":") to the
// message. The length prefix is not counted.
func (format *Runlength) Modulate(msg *core.Message) core.ModulateResult {
	lengthStr := strconv.Itoa(msg.Len())

	dataSize := len(lengthStr) + len(format.separator) + msg.Len()
	payload := core.MessageDataPool.Get(dataSize)

	offset := copy(payload, []byte(lengthStr))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], msg.Data())

	msg.Store(payload)
	return core.ModulateResultContinue
}
