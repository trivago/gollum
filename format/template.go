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
	"bytes"
	"text/template"

	"github.com/trivago/gollum/core"
)

// Template formatter
//
// This formatter allows to apply go templating to a message based on the
// currently set metadata. The template language is described in the go
// documentation: https://golang.org/pkg/text/template/#hdr-Actions
//
// Parameters
//
// - Template: Defines the go template to apply.
// By default this parameter is set to "".
//
// Examples
//
// This example writes the fields "Name" and "Surname" from metadata as
// the new payload.
//
//  exampleProducer:
//    Type: proucer.Console
//    Streams: "*"
//    Modulators:
//      - format.Template:
//        Template: "{{.Name}} {{.Surname}}"
type Template struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	template             *template.Template
}

func init() {
	core.TypeRegistry.Register(Template{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Template) Configure(conf core.PluginConfigReader) {
	var err error
	tpl := conf.GetString("Template", "")
	format.template, err = template.New("Template").Parse(tpl)
	conf.Errors.Push(err)
}

// ApplyFormatter update message payload
func (format *Template) ApplyFormatter(msg *core.Message) error {
	values, err := format.GetSourceAsMetadata(msg)
	if err != nil {
		return err
	}

	templateData := bytes.Buffer{}
	if err = format.template.Execute(&templateData, values); err != nil {
		return err
	}

	format.SetTargetData(msg, templateData.String())
	return nil
}
