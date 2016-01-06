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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tstrings"
)

// Envelope is a formatter that allows prefixing and/or postfixing a message
// with configurable strings.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Envelope"
//     EnvelopeFormatter: "format.Forward"
//     EnvelopePrefix: "<data>"
//     EnvelopePostfix: "</data>\n"
//
// Prefix defines the message prefix. By default this is set to "".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
//
// Postfix defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
//
// EnvelopeDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Envelope struct {
	base    core.Formatter
	postfix string
	prefix  string
}

func init() {
	tgo.TypeRegistry.Register(Envelope{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Envelope) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("EnvelopeFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	format.prefix = tstrings.Unescape(conf.GetString("EnvelopePrefix", ""))
	format.postfix = tstrings.Unescape(conf.GetString("EnvelopePostfix", "\n"))

	return nil
}

// Format adds prefix and postfix to the message formatted by the base formatter
func (format *Envelope) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	basePayload, streamID := format.base.Format(msg)

	prefixLen := len(format.prefix)
	baseLen := len(basePayload)
	postfixLen := len(format.postfix)

	payload := make([]byte, prefixLen+baseLen+postfixLen)

	if prefixLen > 0 {
		prefixLen = copy(payload, format.prefix)
	}

	baseLen = copy(payload[prefixLen:], basePayload)

	if postfixLen > 0 {
		copy(payload[prefixLen+baseLen:], format.postfix)
	}

	return payload, streamID
}
