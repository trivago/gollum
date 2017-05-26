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
	"encoding/json"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// ExtractJSON formatter plugin
// ExtractJSON is a formatter that extracts a single value from a JSON
// message.
// Configuration example
//
//  - format.ExtractJSON:
//      Field: ""
//      TrimValues: true
//      Precision: 0
//      ApplyTo: "payload" # payload or <metaKey>
//
// ExtractJSONDataFormatter formatter that will be applied before
// the field is extracted. Set to format.Forward by default.
//
// ExtractJSONField defines the field to extract. This value is empty by
// default. If the field does not exist an empty string is returned.
//
// ExtractJSONTrimValues will trim whitspaces from the value if enabled.
// Enabled by default.
//
// ExtractJSONPrecision defines the floating point precision of number
// values. By default this is set to 0 i.e. all decimal places will be
// omitted.
type ExtractJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	field                string
	trimValues           bool
	numberFormat         string
	applyTo              string
}

func init() {
	core.TypeRegistry.Register(ExtractJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ExtractJSON) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.field = conf.GetString("Field", "")
	format.trimValues = conf.GetBool("TrimValues", true)
	precision := conf.GetInt("Precision", 0)
	format.numberFormat = fmt.Sprintf("%%.%df", precision)
	format.applyTo = conf.GetString("ApplyTo", core.ApplyToPayloadString)

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *ExtractJSON) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)

	value, err := format.extractJSON(content)
	if err != nil {
		return err
	}

	if value != nil {
		format.SetAppliedContent(msg, value)
	} else {
		if format.applyTo == core.ApplyToPayloadString {
			msg.ResizePayload(0)
		} else {
			msg.GetMetadata().ResetValue(format.applyTo)
		}
	}

	return nil
}

func (format *ExtractJSON) extractJSON(content []byte) ([]byte, error) {
	values := tcontainer.NewMarshalMap()

	err := json.Unmarshal(content, &values)
	if err != nil {
		format.Log.Warning.Print("ExtractJSON failed to unmarshal a message: ", err)
		return nil, err
	}

	if value, exists := values[format.field]; exists {
		switch value.(type) {
		case int64:
			val, _ := value.(int64)
			return []byte(fmt.Sprintf("%d", val)), nil
		case string:
			val, _ := value.(string)
			return []byte(val), nil
		case float64:
			val, _ := value.(float64)
			return []byte(fmt.Sprintf(format.numberFormat, val)), nil
		default:
			return []byte(fmt.Sprintf("%v", value)), nil
		}
	}

	format.Log.Warning.Print("ExtractJSON field not exists: ", format.field)

	return nil, nil
}
