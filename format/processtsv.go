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
	"strconv"
	"strings"
	"time"

	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tstrings"
)

// ProcessTSV formatter plugin
// ProcessTSV is a formatter that allows modifications to fields of a given
// TSV message. The message is modified and returned again as TSV.
// Configuration example
//
//  - format.processTSV:
//      Delimiter: '\t'
//      QuotedValues: false
//      Directives:
//        - "0:time:20060102150405:2006-01-02 15\\:04\\:05"
//        - "3:replace:°:\n"
//        - "6:prefix:0."
//        - "6:postfix:000"
//        - "7:trim: "
//        - "10:quote"
//        - "11:remove"
//        - "11:agent:browser:os:version"
//
// Directives defines the action to be applied to the tsv payload.
// Directives are processed in order of appearance.
// The directives have to be given in the form of key:operation:parameters, where
// operation can be one of the following.
//  * `replace:<old>:<new>` replace a given string in the value with a new one.
//  * `prefix:<string>` add a given string to the start of the value.
//  * `postfix:<string>` add a given string to the end of the value.
//  * `trim:<characters>` remove the given characters (not string!) from the start
//    and end of the value.
//  * quote add a " to the start and end of the value after processing.
//  * `timestamp:<read>:<write>` read a timestamp and transform it into another
//    format.
//  * remove remove the value.
//  * `agent{:<user_agent_field>:<user_agent_field>:...}` Parse the value as a user
//    agent string and extract the given fields into <key>_<user_agent_field>
//    ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os").
//
// Delimiter defines what value separator to split on. Defaults to tabs.
//
// QuotedValue defines if a value that starts and ends with " may
// contain ProcessTSVDelimiter without being split. Default is false.
//
type ProcessTSV struct {
	core.SimpleFormatter
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
	core.TypeRegistry.Register(ProcessTSV{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ProcessTSV) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	directives := conf.GetStringArray("Directives", []string{})

	format.directives = make([]tsvDirective, 0, len(directives))
	format.delimiter = conf.GetString("Delimiter", "\t")
	format.quotedValues = conf.GetBool("QuotedValues", false)

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
				newDirective.parameters = append(newDirective.parameters, tstrings.Unescape(parts[i]))
			}

			format.directives = append(format.directives, newDirective)
		}
	}

	return conf.Errors.OrNil()
}

func stringsToTSVValues(values []string) []tsvValue {
	tsvValues := make([]tsvValue, len(values))
	for index, value := range values {
		tsvValues[index] = tsvValue{
			quoted: false,
			value:  value,
		}
	}
	return tsvValues
}

func (format *ProcessTSV) processTSVDirective(directive tsvDirective, values []tsvValue) []tsvValue {
	if len(values) > directive.index {
		value := values[directive.index].value

		numParameters := len(directive.parameters)
		switch directive.operation {
		case "time":
			if numParameters == 2 {

				if timestamp, err := time.Parse(directive.parameters[0], value[:len(directive.parameters[0])]); err != nil {
					format.Log.Warning.Print("ProcessTSV failed to parse a timestamp: ", err)
				} else {
					values[directive.index].value = timestamp.Format(directive.parameters[1])
				}
			}

		case "replace":
			if numParameters == 2 {
				values[directive.index].value = strings.Replace(value, tstrings.Unescape(directive.parameters[0]), tstrings.Unescape(directive.parameters[1]), -1)
			}

		case "prefix":
			if numParameters == 1 {
				values[directive.index].value = tstrings.Unescape(directive.parameters[0]) + value
			}

		case "postfix":
			if numParameters == 1 {
				values[directive.index].value = value + tstrings.Unescape(directive.parameters[0])
			}

		case "trim":
			switch {
			case numParameters == 0:
				values[directive.index].value = strings.Trim(value, " ")
			case numParameters == 1:
				values[directive.index].value = strings.Trim(value, tstrings.Unescape(directive.parameters[0]))
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

// ApplyFormatter update message payload
func (format *ProcessTSV) ApplyFormatter(msg *core.Message) error {
	if len(format.directives) == 0 {
		return nil // continue
	}

	values := make([]tsvValue, 0)
	if format.quotedValues {
		remainder := string(msg.Data())
		for true {
			if remainder[:1] != `"` {
				split := strings.SplitN(remainder, format.delimiter+`"`, 2)
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
			split := strings.SplitN(remainder, `"`+format.delimiter, 2)
			if len(split) == 2 {
				values = append(values, tsvValue{
					quoted: true,
					value:  split[0],
				})
				remainder = split[1]
			} else {
				if split[0][len(split[0])-1:] != `"` {
					// unmatched quote, abort processing this message
					format.Log.Warning.Print("ProcessTSV failed to parse a message: unmatched quote")
					return nil // continue
				}
				values = append(values, tsvValue{
					quoted: true,
					value:  split[0][:len(split[0])-1],
				})
				break
			}
		}
	} else {
		values = stringsToTSVValues(strings.Split(string(msg.Data()), format.delimiter))
	}

	for _, directive := range format.directives {
		values = format.processTSVDirective(directive, values)
	}

	stringValues := make([]string, len(values))
	for index, value := range values {
		if value.quoted {
			stringValues[index] = `"` + value.value + `"`
		} else {
			stringValues[index] = value.value
		}
	}

	msg.Store([]byte(strings.Join(stringValues, format.delimiter)))
	return nil // continue
}
