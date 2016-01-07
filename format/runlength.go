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
	"strconv"
)

// Runlength is a formatter that prepends the length of the message, followed by
// a ":". The actual message is formatted by a nested formatter.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Runlength"
//     RunlengthFormatter: "format.Envelope"
//
// RunlengthDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Runlength struct {
	base core.Formatter
}

func init() {
	core.TypeRegistry.Register(Runlength{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Runlength) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("RunlengthFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the length of the message (followed by ":") to the message.
func (format *Runlength) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	basePayload, stream := format.base.Format(msg)
	baseLength := len(basePayload)
	lengthStr := strconv.Itoa(baseLength) + ":"

	payload := make([]byte, len(lengthStr)+baseLength)
	len := copy(payload, []byte(lengthStr))
	copy(payload[len:], basePayload)

	return payload, stream
}
