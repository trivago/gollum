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
	"regexp"
)

// RegExp is a formatter that allows parsing a message via regular expression and
// returning the results.
//
//   - "<producer|stream>":
//     Formatter: "format.RegExp"
//     Expression: "(.*)"
//     Template: "${1}"
//     Posix: true
type RegExp struct {
	core.SimpleFormatter `gollumdoc:embed_type`
	expression *regexp.Regexp
	template   []byte
}

func init() {
	core.TypeRegistry.Register(RegExp{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *RegExp) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	var err error
	if conf.GetBool("Posix", true) {
		format.expression, err = regexp.CompilePOSIX(conf.GetString("Expression", "(.*)"))
	} else {
		format.expression, err = regexp.Compile(conf.GetString("Expression", "(.*)"))
	}

	conf.Errors.Push(err)
	format.template = []byte(conf.GetString("Template", "${1}"))

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *RegExp) ApplyFormatter(msg *core.Message) error {
	matches := format.expression.FindSubmatchIndex(msg.Data())
	transformed := format.expression.Expand([]byte{}, format.template, msg.Data(), matches)

	msg.Store(transformed)
	return nil
}
