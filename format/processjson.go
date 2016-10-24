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
	"encoding/json"
	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strconv"
	"strings"
	"time"
)

// ProcessJSON formatter plugin
// ProcessJSON is a formatter that allows modifications to fields of a given
// JSON message. The message is modified and returned again as JSON.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.ProcessJSON"
//    ProcessJSONDataFormatter: "format.Forward"
//    ProcessJSONDirectives:
//      - "host:split: :host:@timestamp"
//      - "@timestamp:time:20060102150405:2006-01-02 15\\:04\\:05"
//      - "error:replace:°:\n"
//      - "text:trim: \t"
//      - "foo:rename:bar"
//		- "foobar:remove"
//      - "user_agent:agent:browser:os:version"
//    ProcessJSONTrimValues: true
//
// ProcessJSONDataFormatter formatter that will be applied before
// ProcessJSONDirectives are processed.
//
// ProcessJSONDirectives defines the action to be applied to the json payload.
// Directives are processed in order of appearance.
// The directives have to be given in the form of key:operation:parameters, where
// operation can be one of the following.
//  * split:<string>:{<key>:<key>:...} Split the value by a string and set the
//    resulting array elements to the given fields in order of appearance.
//  * replace:<old>:<new> replace a given string in the value with a new one
//  * trim:<characters> remove the given characters (not string!) from the start
//    and end of the value
//  * rename:<old>:<new> rename a given field
//  * remove remove a given field
//  * time:<read>:<write> read a timestamp and transform it into another
//    format
//  * unixtimestamp:<read>:<write> read a unix timestamp and transform it into another
//    format. valid read formats are s, ms, and ns.
//  * agent:{<user_agent_field>:<user_agent_field>:...} Parse the value as a user
//    agent string and extract the given fields into <key>_<user_agent_field>
//    ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os")
//  * flatten:{<delimiter>} create new fields from the values in field, with new
//    fields named field + delimiter + subfield. Delimiter defaults to ".".
//    Removes the original field.
//
// ProcessJSONTrimValues will trim whitspaces from all values if enabled.
// Enabled by default.
type ProcessJSON struct {
	base       core.Formatter
	directives []transformDirective
	trimValues bool
}

type transformDirective struct {
	key        string
	operation  string
	parameters []string
}

func init() {
	shared.TypeRegistry.Register(ProcessJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ProcessJSON) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("ProcessJSONDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	directives := conf.GetStringArray("ProcessJSONDirectives", []string{})

	format.base = plugin.(core.Formatter)
	format.directives = make([]transformDirective, 0, len(directives))
	format.trimValues = conf.GetBool("ProcessJSONTrimValues", true)

	for _, directive := range directives {
		directive := strings.Replace(directive, "\\:", "\r", -1)
		parts := strings.Split(directive, ":")
		for i, value := range parts {
			parts[i] = strings.Replace(value, "\r", ":", -1)
		}

		if len(parts) >= 2 {
			newDirective := transformDirective{
				key:       parts[0],
				operation: strings.ToLower(parts[1]),
			}

			for i := 2; i < len(parts); i++ {
				newDirective.parameters = append(newDirective.parameters, shared.Unescape(parts[i]))
			}

			format.directives = append(format.directives, newDirective)
		}
	}

	return nil
}

func processDirective(directive transformDirective, values *shared.MarshalMap) {
	if value, keyExists := (*values)[directive.key]; keyExists {

		numParameters := len(directive.parameters)
		switch directive.operation {
		case "rename":
			if numParameters == 1 {
				(*values)[shared.Unescape(directive.parameters[0])] = value
				delete(*values, directive.key)
			}

		case "time":
			stringValue, err := values.String(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			if numParameters == 2 {

				if timestamp, err := time.Parse(directive.parameters[0], stringValue[:len(directive.parameters[0])]); err != nil {
					Log.Warning.Print("ProcessJSON failed to parse a timestamp: ", err)
				} else {
					(*values)[directive.key] = timestamp.Format(directive.parameters[1])
				}
			}

		case "unixtimestamp":
			floatValue, err := values.Float64(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			intValue := int64(floatValue)
			if numParameters == 2 {
				s, ns := int64(0), int64(0)
				switch directive.parameters[0] {
				case "s":
					s = intValue
				case "ms":
					ns = intValue * int64(time.Millisecond)
				case "ns":
					ns = intValue
				default:
					return
				}
				(*values)[directive.key] = time.Unix(s, ns).Format(directive.parameters[1])
			}

		case "split":
			stringValue, err := values.String(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			if numParameters > 1 {
				token := shared.Unescape(directive.parameters[0])
				if strings.Contains(stringValue, token) {
					elements := strings.Split(stringValue, token)
					mapping := directive.parameters[1:]
					maxItems := shared.MinI(len(elements), len(mapping))

					for i := 0; i < maxItems; i++ {
						(*values)[mapping[i]] = elements[i]
					}
				}
			}

		case "remove":
			if numParameters == 0 {
				delete(*values, directive.key)
			}

		case "replace":
			stringValue, err := values.String(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			if numParameters == 2 {
				(*values)[directive.key] = strings.Replace(stringValue, shared.Unescape(directive.parameters[0]), shared.Unescape(directive.parameters[1]), -1)
			}

		case "trim":
			stringValue, err := values.String(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			switch {
			case numParameters == 0:
				(*values)[directive.key] = strings.Trim(stringValue, " \t")
			case numParameters == 1:
				(*values)[directive.key] = strings.Trim(stringValue, shared.Unescape(directive.parameters[0]))
			}

		case "agent":
			stringValue, err := values.String(directive.key)
			if err != nil {
				Log.Warning.Print(err.Error())
				return
			}
			fields := []string{
				"mozilla",
				"platform",
				"os",
				"localization",
				"engine",
				"engine_version",
				"browser",
				"version",
			}
			if numParameters > 0 {
				fields = directive.parameters
			}
			ua := user_agent.New(stringValue)
			for _, field := range fields {
				switch field {
				case "mozilla":
					(*values)[directive.key+"_mozilla"] = ua.Mozilla()
				case "platform":
					(*values)[directive.key+"_platform"] = ua.Platform()
				case "os":
					(*values)[directive.key+"_os"] = ua.OS()
				case "localization":
					(*values)[directive.key+"_localization"] = ua.Localization()
				case "engine":
					(*values)[directive.key+"_engine"], _ = ua.Engine()
				case "engine_version":
					_, (*values)[directive.key+"_engine_version"] = ua.Engine()
				case "browser":
					(*values)[directive.key+"_browser"], _ = ua.Browser()
				case "version":
					_, (*values)[directive.key+"_version"] = ua.Browser()
				}
			}

		case "flatten":
			delimiter := "."
			if numParameters == 1 {
				delimiter = directive.parameters[0]
			}
			keyPrefix := directive.key + delimiter
			if mapValue, err := values.Map(directive.key); err == nil {
				for key, val := range mapValue {
					(*values)[keyPrefix + key.(string)] = val
				}
			} else if arrayValue, err := values.Array(directive.key); err == nil {
				for index, val := range arrayValue {
					(*values)[keyPrefix + strconv.Itoa(index)] = val
				}
			} else {
				Log.Warning.Print("key was not a JSON array or object: " + directive.key)
				return
			}
			delete(*values, directive.key)
		}
	}
}

// Format modifies the JSON payload of this message
func (format *ProcessJSON) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	if len(format.directives) == 0 {
		return data, streamID // ### return, no directives ###
	}

	values := make(shared.MarshalMap)
	err := json.Unmarshal(data, &values)
	if err != nil {
		Log.Warning.Print("ProcessJSON failed to unmarshal a message: ", err)
		return data, streamID // ### return, malformed data ###
	}

	for _, directive := range format.directives {
		processDirective(directive, &values)
	}

	if format.trimValues {
		for key := range values {
			stringValue, err := values.String(key)
			if err == nil {
				values[key] = strings.Trim(stringValue, " ")
			}
		}
	}

	if jsonData, err := json.Marshal(values); err == nil {
		return jsonData, streamID // ### return, ok ###
	}

	Log.Warning.Print("ProcessJSON failed to marshal a message: ", err)
	return data, streamID
}
