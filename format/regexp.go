// Copyright 2015-2018 trivago N.V.
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
	"regexp"
)

// RegExp formatter
//
// This formatter parses a message using a regular expression, performs
// string (template) replacement and returns the result.
//
// Parameters
//
// - Posix: Set to true to compile the regular expression using posix semantics.
// By default this parameter is set to true.
//
// - Expression: Defines the regular expression used for parsing.
// For details on the regexp syntax see https://golang.org/pkg/regexp/syntax.
// By default this parameter is set to "(.*)"
//
// - Template: Defines the result string. Regexp matching groups can be referred
// to using "${n}", with n being the group's index. For other possible
// reference semantics, see https://golang.org/pkg/regexp/#Regexp.Expand.
// By default this parameter is set to "${1}"
//
// Examples
//
// This example extracts time and host from an imaginary log message format.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stding
//    Modulators:
//      - format.RegExp:
//        Expression: "^(\\d+) (\\w+): "
//        Template: "time: ${1}, host: ${2}"
type RegExp struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	expression           *regexp.Regexp
	template             []byte `config:"Template" default:"${1}"`
}

func init() {
	core.TypeRegistry.Register(RegExp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *RegExp) Configure(conf core.PluginConfigReader) {
	var err error
	if conf.GetBool("Posix", true) {
		format.expression, err = regexp.CompilePOSIX(conf.GetString("Expression", "(.*)"))
	} else {
		format.expression, err = regexp.Compile(conf.GetString("Expression", "(.*)"))
	}
	conf.Errors.Push(err)
}

// ApplyFormatter update message payload
func (format *RegExp) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)
	matches := format.expression.FindSubmatchIndex(content)
	transformed := format.expression.Expand([]byte{}, format.template, content, matches)

	format.SetAppliedContent(msg, transformed)
	return nil
}
