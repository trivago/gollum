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
	"github.com/trivago/tgo/tstrings"
)

// Envelope formatter plugin
// Envelope is a formatter that allows prefixing and/or postfixing a message
// with configurable strings.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Envelope"
//    EnvelopeFormatter: "format.Forward"
//    EnvelopePrefix: ""
//    EnvelopePostfix: "\n"
//
// EnvelopePrefix defines the message prefix. By default this is set to "".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
//
// EnvelopePostfix defines the message postfix. By default this is set to "\n".
// Special characters like \n \r \t will be transformed into the actual control
// characters.
//
// EnvelopeFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
type Envelope struct {
	core.SimpleFormatter
	postfix string
	prefix  string
}

func init() {
	core.TypeRegistry.Register(Envelope{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Envelope) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.prefix = tstrings.Unescape(conf.GetString("Prefix", ""))
	format.postfix = tstrings.Unescape(conf.GetString("Postfix", "\n"))
	return conf.Errors.OrNil()
}

// Modulate adds prefix and postfix to the message formatted by the base formatter
func (format *Envelope) Modulate(msg *core.Message) core.ModulateResult {
	prefixLen := len(format.prefix)
	postfixLen := len(format.postfix)
	offset := 0

	payload := core.MessageDataPool.Get(prefixLen + msg.Len() + postfixLen)

	if prefixLen > 0 {
		offset = copy(payload, format.prefix)
	}

	offset += copy(payload[offset:], msg.Data())

	if postfixLen > 0 {
		copy(payload[offset:], format.postfix)
	}

	msg.Store(payload)
	return core.ModulateResultContinue
}
