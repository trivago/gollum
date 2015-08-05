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
)

// Serialize is a formatter that serializes a message for later retrieval.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Serialize"
//     SerializeFormatter: "format.Envelope"
//
// SerializeFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Serialize struct {
	base core.Formatter
}

func init() {
	shared.TypeRegistry.Register(Serialize{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Serialize) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("SerializeFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	return nil
}

// Format prepends the sequence number of the message (followed by ":") to the
// message.
func (format *Serialize) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	var stream core.MessageStreamID
	msg.Data, stream = format.base.Format(msg)
	return []byte(msg.Serialize() + "\n"), stream
}
