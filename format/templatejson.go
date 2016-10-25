// Copyright 2015-2016 trivago GmbH
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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
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
	base         core.Formatter
	template     *template.Template
}

func init() {
	shared.TypeRegistry.Register(TemplateJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *TemplateJSON) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("TemplateJSONFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	templ := conf.GetString("TemplateJSONTemplate", "")
	format.template, err = template.New("TemplateJSON").Parse(templ)
	if err != nil {
		return err
	}

	return nil
}

// Format executes the template against the JSON payload of this message
func (format *TemplateJSON) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)

	values := shared.NewMarshalMap()
	err := json.Unmarshal(data, &values)
	if err != nil {
		Log.Warning.Print("TemplateJSON failed to unmarshal a message: ", err)
		return data, streamID // ### return, malformed data ###
	}

	var templateData bytes.Buffer
	err = format.template.Execute(&templateData, values)
	if err != nil {
		Log.Warning.Print("TemplateJSON failed to template a message: ", err)
		return data, streamID // ### return, malformed data ###
	}

	return templateData.Bytes(), streamID
}
