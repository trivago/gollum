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
	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strconv"
	"strings"
	"time"
)

// ProcessTSV formatter plugin
// ProcessTSV is a formatter that allows modifications to fields of a given
// TSV message. The message is modified and returned again as TSV.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.processTSV"
//    ProcessTSVDataFormatter: "format.Forward"
//    ProcessTSVDelimiter: '\t'
//    ProcessTSVQuotedValues: false
//    ProcessTSVDirectives:
//      - "0:time:20060102150405:2006-01-02 15\\:04\\:05"
//      - "3:replace:°:\n"
//      - "7:trim: "
//      - "11:remove"
//      - "11:agent:browser:os:version"
//
// ProcessTSVDataFormatter formatter that will be applied before
// ProcessTSVDirectives are processed.
//
// ProcessTSVDirectives defines the action to be applied to the tsv payload.
// Directives are processed in order of appearance.
// The directives have to be given in the form of key:operation:parameters, where
// operation can be one of the following:
// - replace:<old>:<new> replace a given string in the value with a new one
// - trim:<characters> remove the given characters (not string!) from the start
// and end of the value
// - timestamp:<read>:<write> read a timestamp and transform it into another
// format
// - remove removes the value
// - agent:{<user_agent_field>:<user_agent_field>:...} Parse the value as a user
// agent string and extract the given fields into <key>_<user_agent_field>
// ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os")
//
// ProcessTSVDelimiter defines what value separator to split on. Defaults to tabs.
//
// ProcessTSVQuotedValue defines if a value that starts and ends with " may
// contain ProcessTSVDelimiter without being split. Default is false.
//
type ProcessTSV struct {
	base         core.Formatter
	directives   []tsvDirective
	delimiter    string
	quotedValues bool
}

type tsvDirective struct {
	index      int
	operation  string
	parameters []string
}

type tsvValue struct {
	quoted bool
	value  string
}

func init() {
	shared.TypeRegistry.Register(ProcessTSV{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ProcessTSV) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("ProcessTSVDataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	directives := conf.GetStringArray("ProcessTSVDirectives", []string{})

	format.base = plugin.(core.Formatter)
	format.directives = make([]tsvDirective, 0, len(directives))
	format.delimiter = conf.GetString("ProcessTSVDelimiter", "\t")
	format.quotedValues = conf.GetBool("ProcessTSVQuotedValues", false)

	for _, directive := range directives {
		directive := strings.Replace(directive, "\\:", "\r", -1)
		parts := strings.Split(directive, ":")
		for i, value := range parts {
			parts[i] = strings.Replace(value, "\r", ":", -1)
		}

		if len(parts) >= 2 {
			index, err := strconv.Atoi(parts[0])
			if err != nil {
				return err
			}

			newDirective := tsvDirective{
				index:     index,
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

func stringsToTSVValues(values []string) ([]tsvValue) {
	tsvValues := make([]tsvValue, len(values))
	for index, value := range values {
		tsvValues[index] = tsvValue{
			quoted: false,
			value:  value,
		}
	}
	return tsvValues
}

func processTSVDirective(directive tsvDirective, values []tsvValue) ([]tsvValue) {
	if len(values) > directive.index {
		value := values[directive.index].value

		numParameters := len(directive.parameters)
		switch directive.operation {
		case "time":
			if numParameters == 2 {

				if timestamp, err := time.Parse(directive.parameters[0], value[:len(directive.parameters[0])]); err != nil {
					Log.Warning.Print("ProcessTSV failed to parse a timestamp: ", err)
				} else {
					values[directive.index].value = timestamp.Format(directive.parameters[1])
				}
			}

		case "replace":
			if numParameters == 2 {
				values[directive.index].value = strings.Replace(value, shared.Unescape(directive.parameters[0]), shared.Unescape(directive.parameters[1]), -1)
			}

		case "prefix":
			if numParameters == 1 {
				values[directive.index].value = shared.Unescape(directive.parameters[0]) + value
			}

		case "postfix":
			if numParameters == 1 {
				values[directive.index].value = value + shared.Unescape(directive.parameters[0])
			}

		case "trim":
			switch {
			case numParameters == 0:
				values[directive.index].value = strings.Trim(value, " ")
			case numParameters == 1:
				values[directive.index].value = strings.Trim(value, shared.Unescape(directive.parameters[0]))
			}

		case "quote":
			values[directive.index].quoted = true

		case "remove":
			values = append(values[:directive.index], values[directive.index+1:]...)

		case "agent":
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
			ua := user_agent.New(value)
			var agentValues []string
			for _, field := range fields {
				switch field {
				case "mozilla":
					agentValues = append(agentValues, ua.Mozilla())
				case "platform":
					agentValues = append(agentValues, ua.Platform())
				case "os":
					agentValues = append(agentValues, ua.OS())
				case "localization":
					agentValues = append(agentValues, ua.Localization())
				case "engine":
					engine, _ := ua.Engine()
					agentValues = append(agentValues, engine)
				case "engine_version":
					_, engineVersion := ua.Engine()
					agentValues = append(agentValues, engineVersion)
				case "browser":
					browser, _ := ua.Browser()
					agentValues = append(agentValues, browser)
				case "version":
					_, version := ua.Browser()
					agentValues = append(agentValues, version)
				}
			}
			tsvAgentValues := stringsToTSVValues(agentValues)
			// insert agentValues after index
			values = append(values[:directive.index+1], append(tsvAgentValues, values[directive.index+1:]...)...)
		}
	}
	return values
}

// Format modifies the TSV payload of this message
func (format *ProcessTSV) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	if len(format.directives) == 0 {
		return data, streamID // ### return, no directives ###
	}

	values := make([]tsvValue, 0)
	if format.quotedValues {
		remainder := string(data)
		for true {
			if remainder[:1] != `"` {
				split := strings.SplitN(remainder, format.delimiter + `"`, 2)
				tsvValues := stringsToTSVValues(strings.Split(split[0], format.delimiter))
				values = append(values, tsvValues...)
				if len(split) == 2 {
					remainder = split[1]
				} else {
					break
				}
			} else {
				remainder = remainder[1:]
			}
			split := strings.SplitN(remainder, `"` + format.delimiter, 2)
			if len(split) == 2 {
				values = append(values, tsvValue{
					quoted: true,
					value:  split[0],
				})
				remainder = split[1]
			} else {
				if split[0][len(split[0])-1:] != `"` {
					// unmatched quote, abort processing this message
					Log.Warning.Print("ProcessTSV failed to parse a message: unmatched quote")
					return data, streamID
				}
				values = append(values, tsvValue{
					quoted: true,
					value:  split[0][:len(split[0])-1],
				})
				break
			}
		}
	} else {
		values = stringsToTSVValues(strings.Split(string(data), format.delimiter))
	}

	for _, directive := range format.directives {
		values = processTSVDirective(directive, values)
	}

	stringValues := make([]string, len(values))
	for index, value := range values {
		if value.quoted {
			stringValues[index] = `"` + value.value + `"`
		} else {
			stringValues[index] = value.value
		}
	}

	return []byte(strings.Join(stringValues, format.delimiter)), streamID
}
