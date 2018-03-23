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
	"strconv"
	"strings"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// ExtractJSON formatter
//
// This formatter extracts a specific value from a JSON payload and writes it
// back as a new payload or as a metadata field.
//
// Parameters
//
// - Field: Defines the JSON key to extract. If the field does not exist an
// empty string is returned. Field paths can be defined in a format accepted by
// tgo.MarshalMap.Path.
// By default this parameter is set to "".
//
// - TrimValues: Enables trimming of whitespaces at the beginning and end of the
// extracted value.
// By default this parameter is set to true.
//
// - Precision: Defines the number of decimal places to use when converting
// Numbers into strings. If this parameter is set to -1 the shortest possible
// number of decimal places will be used.
// By default this parameter is set to -1.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - formatter.ExtractJSON
//        Field: host
//        ApplyTo: host
type ExtractJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	field                string `config:"Field"`
	precision            int    `config:"Precision" default:"-1"`
	trimValues           bool   `config:"TrimValues" default:"true"`
}

func init() {
	core.TypeRegistry.Register(ExtractJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ExtractJSON) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *ExtractJSON) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)

	value, err := format.extractJSON(content)
	if err != nil {
		return err
	}

	format.SetAppliedContent(msg, value)
	return nil
}

func (format *ExtractJSON) extractJSON(content []byte) ([]byte, error) {
	values := tcontainer.NewMarshalMap()

	err := json.Unmarshal(content, &values)
	if err != nil {
		format.Logger.Warning("ExtractJSON failed to unmarshal a message: ", err)
		return nil, err
	}

	var strValue string

	if value, exists := values[format.field]; exists {
		switch value.(type) {
		case int64:
			val, _ := value.(int64)
			strValue = strconv.FormatInt(val, 10)

		case string:
			strValue, _ = value.(string)

		case float64:
			val, _ := value.(float64)
			strValue = strconv.FormatFloat(val, 'f', format.precision, 64)

		default:
			strValue = fmt.Sprintf("%v", value)
		}

		if format.trimValues {
			return []byte(strings.TrimSpace(strValue)), nil
		}
	}

	format.Logger.Warning("ExtractJSON field not exists: ", format.field)

	return nil, nil
}
