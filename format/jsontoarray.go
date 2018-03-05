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
	"encoding/json"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// JSONToArray formatter plugin
//
// JSONToArray "flattens" a JSON object by selecting specific fields
// and creating a delimiter-separated string of their values.
//
// A JSON input like `{"foo":"value1","bar":"value2"}` can be transformed
// into a list like `value1,value2`.
//
// Parameters
//
// - Fields: List of strings specifying the JSON keys to retrieve from the input
//
// - Separator: The delimited string to insert between each value in the generated
// string.
// By default this parameter is set to ",".
//
// Examples
//
// This example get the `foo` and `bar` fields from a json document
// and create a payload of `foo_value:bar_value`:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.JSONToArray
//        Separator: ;
//        Fields:
//          - foo
//          - bar
type JSONToArray struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            string   `config:"Separator" default:","`
	fields               []string `config:"Fields"`
}

func init() {
	core.TypeRegistry.Register(JSONToArray{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSONToArray) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *JSONToArray) ApplyFormatter(msg *core.Message) error {
	content, err := format.getCsvContent(format.GetAppliedContent(msg))
	if err != nil {
		return err
	}

	format.SetAppliedContent(msg, content)
	return nil
}

func (format *JSONToArray) getCsvContent(content []byte) ([]byte, error) {
	values := make(tcontainer.MarshalMap)
	err := json.Unmarshal(content, &values)
	if err != nil {
		format.Logger.Error("Json parsing error: ", err)
		return nil, err
	}

	csv := ""
	for _, field := range format.fields {
		if value, exists := values.Value(field); exists {
			// Add value to csv string based on the actual type
			switch value.(type) {
			case bool:
				csv = fmt.Sprintf("%s%t%s", csv, value.(bool), format.separator)
			case float64:
				csv = fmt.Sprintf("%s%d%s", csv, value.(int), format.separator)
			case string:
				csv = fmt.Sprintf("%s%s%s", csv, value.(string), format.separator)
			default:
				format.Logger.Warning("Field ", field, " uses an unsupported datatype")
				csv = format.separator
			}
		} else {
			// Not parsable = empty value
			format.Logger.Warning("Field ", field, " not found")
			csv += format.separator
		}
	}
	// Remove last separator
	if len(csv) >= len(format.separator) {
		csv = csv[:len(csv)-len(format.separator)]
	}

	return []byte(csv), nil
}
