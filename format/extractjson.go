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
//  - "stream.Broadcast":
//    Formatter: "format.ExtractJSON"
//    ExtractJSONdataFormatter: "format.Forward"
//    ExtractJSONField: ""
//    ExtractJSONTrimValues: true
//    ExtractJSONPrecision: 0
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
	core.SimpleFormatter
	field        string
	trimValues   bool
	numberFormat string
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

	return conf.Errors.OrNil()
}

// Modulate extractes a given field from a JSON message. If the payload is no
// valid JSON the message will be discarded.
func (format *ExtractJSON) Modulate(msg *core.Message) core.ModulateResult {
	values := tcontainer.NewMarshalMap()
	err := json.Unmarshal(msg.Data(), &values)
	if err != nil {
		format.Log.Warning.Print("ExtractJSON failed to unmarshal a message: ", err)
		return core.ModulateResultDiscard // ### return, malformed data ###
	}

	if value, exists := values[format.field]; exists {
		switch value.(type) {
		case int64:
			val, _ := value.(int64)
			msg.Store([]byte(fmt.Sprintf("%d", val)))
		case string:
			val, _ := value.(string)
			msg.Store([]byte(val))
		case float64:
			val, _ := value.(float64)
			msg.Store([]byte(fmt.Sprintf(format.numberFormat, val)))
		default:
			msg.Store([]byte(fmt.Sprintf("%v", value)))
		}
	} else {
		msg.Resize(0)
	}

	return core.ModulateResultContinue
}
