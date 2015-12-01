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
	"encoding/json"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strings"
)

// ProcessJSON is a formatter that allows modifications to fields of a given
// JSON message. The message is modified and returned again as JSON.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.processJSON"
//     ProcessJSONDataFormatter: "format.Forward"
//     ProcessJSONDirectives:
//       "host": "split: :host:@timestamp"
//       "error": "replace:°:\n"
//       "text": "trim:a:b:c"
//	   ProcessJSONTrimFields: true
//
// ProcessJSONDataFormatter formatter that will be applied before
// ProcessJSONDirectives are processed.
//
// ProcessJSONDirectives defines the action to be applied to a given member.
// The directives have to be given in the form of field:operation, where
// operation can be one of the following:
// - split:<string>:[<key>:<key>:...] Split the value by a string and set the
// resulting array elements to the given fields in order of appearance.
// - replace:<old>:<new> replace a given string in the value with a new one
// - trim:<characters> remove the given characters (not string!) from the start
// and end of the value
//
// ProcessJSONTrimValues will trim whitspaces from all values if enabled.
// Enabled by default.
type ProcessJSON struct {
	base       core.Formatter
	directives map[string][]string
	trimValues bool
}

type valueMap map[string]string

func init() {
	shared.TypeRegistry.Register(ProcessJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ProcessJSON) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("ProcessJSONDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.base = plugin.(core.Formatter)
	format.directives = make(map[string][]string)
	format.trimValues = conf.GetBool("ProcessJSONTrimValues", true)

	directives := conf.GetStringMap("ProcessJSONDirectives", make(map[string]string))
	for key, directive := range directives {
		directive := strings.Replace(directive, "\\:", "\r", -1)
		parts := strings.Split(directive, ":")
		for i, part := range parts {
			parts[i] = strings.Replace(part, "\r", ":", -1)
		}
		format.directives[key] = parts

	}
	return nil
}

func (values *valueMap) processDirective(key string, parameters []string) {
	if value, keyExists := (*values)[key]; keyExists {

		numParameters := len(parameters)
		switch strings.ToLower(parameters[0]) {
		case "split":
			if numParameters > 1 {
				token := shared.Unescape(parameters[1])
				if strings.Contains(value, token) {
					elements := strings.Split(value, token)
					mapping := parameters[2:]
					maxItems := shared.MinI(len(elements), len(mapping))

					for i := 0; i < maxItems; i++ {
						(*values)[mapping[i]] = elements[i]
					}
				}
			}

		case "replace":
			if numParameters == 3 {
				(*values)[key] = strings.Replace(value, shared.Unescape(parameters[1]), shared.Unescape(parameters[2]), -1)
			}

		case "trim":
			switch {
			case numParameters == 1:
				(*values)[key] = strings.Trim(value, " \t")
			case numParameters == 2:
				(*values)[key] = strings.Trim(value, shared.Unescape(parameters[1]))
			}
		}
	}
}

// Format modifies the JSON payload of this message
func (format *ProcessJSON) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	if len(format.directives) == 0 {
		return data, streamID // ### return, no directives ###
	}

	values := make(valueMap)
	err := json.Unmarshal(data, &values)
	if err != nil {
		Log.Warning.Print("ProcessJSON failed to unmarshal a message: ", err)
		return data, streamID // ### return, malformed data ###
	}

	for key, parameters := range format.directives {
		values.processDirective(key, parameters)
	}

	if format.trimValues {
		for key, value := range values {
			values[key] = strings.Trim(value, " ")
		}
	}

	if jsonData, err := json.Marshal(values); err == nil {
		return jsonData, streamID // ### return, ok ###
	}

	Log.Warning.Print("ProcessJSON failed to marshal a message: ", err)
	return data, streamID
}
