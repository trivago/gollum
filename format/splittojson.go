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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
)

// SplitToJSON formatter plugin
//
// SplitToJSON is a formatter that splits a message by a given token and creates
// a JSON object of the split values by assigning each value to a predefined property.
//
// Parameters
//
// - Keys: This value defines an array of JSON keys to which the split message's parts
// should be assigned to. The keys are applied to the resulting token array by index.
//
// - SplitBy: This value defines the separator character to use when splitting a message.
// By default this parameter is set to "|".
//
// - KeepJSON: This value can be set to "false" to escape JSON payload texts
// as regualar strings. Otherwise JSON payload will be taken as-is and set to the
// corresponding key.
// By default this parameter is set to "true".
//
// Examples
//
// This example will format a input of `value1,value2,value3` to a json
// string of `{"foo":"value1", "bar":"value2"}`:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.SplitToJSON:
//        SplitBy: ","
//        Keys:
//          - foo
//          - bar
type SplitToJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	token                []byte   `config:"SplitBy" default:"|"`
	keys                 []string `config:"Keys"`
	keepJSON             bool     `config:"KeepJSON" default:"true"`
}

func init() {
	core.TypeRegistry.Register(SplitToJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *SplitToJSON) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *SplitToJSON) ApplyFormatter(msg *core.Message) error {
	components := bytes.Split(format.GetAppliedContent(msg), format.token)
	maxIdx := tmath.MinI(len(format.keys), len(components))
	jsonData := ""

	switch {
	case maxIdx == 0:
	case maxIdx == 1:
		jsonData = fmt.Sprintf("{%s:\"%s\"}", format.keys[0], components[0])
	default:
		for i := 0; i < maxIdx; i++ {
			key := tstrings.EscapeJSON(format.keys[i])
			value := string(components[i])
			if isJSON, _, _ := tstrings.IsJSON(components[i]); !format.keepJSON || !isJSON {
				value = "\"" + tstrings.EscapeJSON(value) + "\""
			}

			switch {
			case i == 0:
				jsonData = fmt.Sprintf("{\"%s\":%s", key, value)
			case i == maxIdx-1:
				jsonData = fmt.Sprintf("%s,\"%s\":%s}", jsonData, key, value)
			default:
				jsonData = fmt.Sprintf("%s,\"%s\":%s", jsonData, key, value)
			}
		}
	}

	format.SetAppliedContent(msg, []byte(jsonData))
	return nil
}
