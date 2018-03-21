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
	"encoding/json"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"text/template"
)

// TemplateJSON formatter
//
// This formatter unmarshals the given data as JSON and applies the results to
// the given go template. The JSON data will be replaced with the rendered
// template result. The template language is described in the go documentation:
// https://golang.org/pkg/text/template/#hdr-Actions
//
// Parameters
//
// - Template: Defines the go template to execute with the received JSON data.
// If the template cannot be parsed or the JSON payload cannot be unmarshaled,
// the incoming JSON data is preserved.
// By default this parameter is set to "".
//
// Examples
//
// This example extracts the fields "Name" and "Surname" from a JSON encoded
// payload and writes them both back as a plain text result.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.TemplateJSON:
//        Template: "{{.Name}} {{.Surname}}"
type TemplateJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	template             *template.Template
}

func init() {
	core.TypeRegistry.Register(TemplateJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *TemplateJSON) Configure(conf core.PluginConfigReader) {
	var err error
	tpl := conf.GetString("Template", "")
	format.template, err = template.New("Template").Parse(tpl)
	conf.Errors.Push(err)
}

// ApplyFormatter update message payload
func (format *TemplateJSON) ApplyFormatter(msg *core.Message) error {
	values := tcontainer.NewMarshalMap()
	err := json.Unmarshal(format.GetAppliedContent(msg), &values)
	if err != nil {
		format.Logger.Warning("TemplateJSON failed to unmarshal a message: ", err)
		return err
	}

	var templateData bytes.Buffer
	err = format.template.Execute(&templateData, values)
	if err != nil {
		format.Logger.Warning("TemplateJSON failed to template a message: ", err)
		return err
	}

	format.SetAppliedContent(msg, templateData.Bytes())

	return nil
}
