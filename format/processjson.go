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
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
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
// operation can be one of the following:
// - split:<string>:{<key>:<key>:...} Split the value by a string and set the
// resulting array elements to the given fields in order of appearance.
// - replace:<old>:<new> replace a given string in the value with a new one
// - trim:<characters> remove the given characters (not string!) from the start
// and end of the value
// - rename:<old>:<new> rename a given field
// - remove remove a given field
// - timestamp:<read>:<write> read a timestamp and transform it into another
// format
// - agent:{<user_agent_field>:<user_agent_field>:...} Parse the value as a user
// agent string and extract the given fields into <key>_<user_agent_field>
// ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os")
//
// ProcessJSONTrimValues will trim whitspaces from all values if enabled.
// Enabled by default.
type ProcessJSON struct {
	core.SimpleFormatter
	directives []transformDirective
	trimValues bool
}

type transformDirective struct {
	key        string
	operation  string
	parameters []string
}

func init() {
	core.TypeRegistry.Register(ProcessJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ProcessJSON) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.trimValues = conf.GetBool("Trim", true)
	directives := conf.GetStringArray("Directives", []string{})

	if len(directives) > 0 {
		format.directives = make([]transformDirective, 0, len(directives))
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
					newDirective.parameters = append(newDirective.parameters, tstrings.Unescape(parts[i]))
				}

				format.directives = append(format.directives, newDirective)
			}
		}
	}

	return conf.Errors.OrNil()
}

func (format *ProcessJSON) processDirective(directive transformDirective, values *tcontainer.MarshalMap) {
	if value, keyExists := (*values)[directive.key]; keyExists {

		numParameters := len(directive.parameters)
		switch directive.operation {
		case "rename":
			if numParameters == 1 {
				(*values)[tstrings.Unescape(directive.parameters[0])] = value
				delete(*values, directive.key)
			}

		case "time":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
				return
			}

			if numParameters == 2 {
				if timestamp, err := time.Parse(directive.parameters[0], stringValue[:len(directive.parameters[0])]); err != nil {
					format.Log.Warning.Print("ProcessJSON failed to parse a timestamp: ", err)
				} else {
					(*values)[directive.key] = timestamp.Format(directive.parameters[1])
				}
			}

		case "split":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
				return
			}
			if numParameters > 1 {
				token := tstrings.Unescape(directive.parameters[0])
				if strings.Contains(stringValue, token) {
					elements := strings.Split(stringValue, token)
					mapping := directive.parameters[1:]
					maxItems := tmath.MinI(len(elements), len(mapping))

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
				format.Log.Warning.Print(err.Error())
				return
			}
			if numParameters == 2 {
				(*values)[directive.key] = strings.Replace(stringValue, tstrings.Unescape(directive.parameters[0]), tstrings.Unescape(directive.parameters[1]), -1)
			}

		case "trim":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
				return
			}
			switch {
			case numParameters == 0:
				(*values)[directive.key] = strings.Trim(stringValue, " \t")
			case numParameters == 1:
				(*values)[directive.key] = strings.Trim(stringValue, tstrings.Unescape(directive.parameters[0]))
			}

		case "agent":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
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
		}
	}
}

// Format modifies the JSON payload of this message
func (format *ProcessJSON) Format(msg *core.Message) {
	if len(format.directives) == 0 {
		return // ### return, no directives ###
	}

	values := make(tcontainer.MarshalMap)
	err := json.Unmarshal(msg.Data(), &values)
	if err != nil {
		format.Log.Warning.Print("ProcessJSON failed to unmarshal a message: ", err)
		return // ### return, malformed data ###
	}

	for _, directive := range format.directives {
		format.processDirective(directive, &values)
	}

	if format.trimValues {
		for key := range values {
			stringValue, err := values.String(key)
			if err == nil {
				values[key] = strings.Trim(stringValue, " ")
			}
		}
	}

	if jsonData, err := json.Marshal(values); err != nil {
		format.Log.Warning.Print("ProcessJSON failed to marshal a message: ", err)
	} else {
		msg.Store(jsonData)
	}
}
