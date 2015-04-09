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
	"strings"
)

// Envelope is a formatter that allows prefixing and/or postfixing a message
// with configurable strings.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Envelope"
//     EnvelopeDataFormatter: "format.Forward"
//     Prefix: "<data>"
//     Postfix: "</data>\n"
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
	core.FormatterBase
	base    core.Formatter
	postfix string
	prefix  string
}

var envelopeEscapeChars = strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

func init() {
	shared.RuntimeType.Register(Envelope{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Envelope) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("EnvelopeDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	format.prefix = envelopeEscapeChars.Replace(conf.GetString("Prefix", ""))
	format.postfix = envelopeEscapeChars.Replace(conf.GetString("Postfix", "\n"))

	return nil
}

// PrepareMessage sets the message to be formatted.
func (format *Envelope) PrepareMessage(msg core.Message) {
	format.base.PrepareMessage(msg)

	prefixLen := len(format.prefix)
	baseLen := format.base.Len()
	postfixLen := len(format.postfix)

	format.FormatterBase.Message = make([]byte, prefixLen+baseLen+postfixLen)

	if prefixLen > 0 {
		prefixLen = copy(format.FormatterBase.Message, format.prefix)
	}

	baseLen, _ = format.base.Read(format.FormatterBase.Message[prefixLen:])

	if postfixLen > 0 {
		copy(format.FormatterBase.Message[prefixLen+baseLen:], format.postfix)
	}
}
