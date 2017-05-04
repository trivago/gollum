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
	"bytes"
	"encoding/json"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"text/template"
)

// TemplateJSON formatter plugin
// TemplateJSON is a formatter that evaluates a text template with an input
// of a JSON message.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.TemplateJSON"
//    TemplateJSONFormatter: "format.Forward"
//    TemplateJSONTemplate: ""
//
// TemplateJSONFormatter formatter that will be applied before
// the field is templated. Set to format.Forward by default.
//
// TemplateJSONTemplate defines the template to execute with text/template.
// This value is empty by default. If the template fails to execute the output
// of TemplateJSONFormatter is returned.
type TemplateJSON struct {
	core.SimpleFormatter
	template *template.Template
}

func init() {
	core.TypeRegistry.Register(TemplateJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *TemplateJSON) Configure(conf core.PluginConfigReader) error {
	var err error

	templ := conf.GetString("TemplateJSONTemplate", "")
	format.template, err = template.New("TemplateJSON").Parse(templ)
	conf.Errors.Push(err)

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *TemplateJSON) ApplyFormatter(msg *core.Message) error {
	values := tcontainer.NewMarshalMap()
	err := json.Unmarshal(msg.Data(), &values)
	if err != nil {
		format.Log.Warning.Print("TemplateJSON failed to unmarshal a message: ", err)
		return err
	}

	var templateData bytes.Buffer
	err = format.template.Execute(&templateData, values)
	if err != nil {
		format.Log.Warning.Print("TemplateJSON failed to template a message: ", err)
		return err
	}
	msg.Store(templateData.Bytes())

	return nil
}
