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
	"strconv"
	"strings"
	"time"

	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tstrings"
)

// ProcessTSV formatter
//
// This formatter allows modification of TSV encoded data. Each field can be
// processed by different directives and the result of all directives will be
// stored back to the original location.
//
// Parameters
//
// - Delimiter: Defines the separator used to split values.
// By default this parameter is set to "\t".
//
// - QuotedValue: When set to true, values that start and end with a quotation
// mark are not scanned for delimiter characters. I.e. those values will not be
// split even if they contain delimiter characters.
// By default this parameter is set to false.
//
// - Directives: Defines an array of actions to apply to the TSV encoded
// data. Directives are processed in order of appearance. Directives start
// with the index of the field, followed by an action followed by additional
// parameters if necessary. Parameters, key and action are separated by
// the ":" character.
// By default this parameter is set to an empty list.
//
//  - replace: <string>  <new string>
//  Replace a given string inside the field's value with another one.
//
//  - prefix: <string>
//  Prepend the given string to the field's value
//
//  - postfix: <string>
//  Append the given string to the field's value
//
//  - trim: <characters>
//  Remove the given characters from the start and end of the field's value.
//
//  - quote:
//  Surround the field's value with quotation marks after all directives have been
//  processed.
//
//  - time: <from fromat> <to format>
//  Read a timestamp in the specified time.Parse-compatible format and transform
//  it into another format compatible with time.Format.
//
//  - remove
//  Removes the field from the result
//
//  - agent: {<field>, <field>, ...}
//  Parse the field's value as a user agent string and insert the given fields
//  into the TSV after the given index.
//  If no fields are given all fields are returned.
//
//   - mozilla: mozilla version
//   - platform: the platform used
//   - os: the operating system used
//   - localization: the language used
//   - engine: codename of the browser engine
//   - engine_version: version of the browser engine
//   - browser: name of the browser
//   - version: version of the browser
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.processTSV:
//        Delimiter: ","
//        Directives:
//          - "0:time:20060102150405:2006-01-02 15\\:04\\:05"
//          - "2:remove"
//          - "11:agent:os:engine:engine_version"
type ProcessTSV struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	directives           []tsvDirective
	delimiter            string `config:"Delimiter" default:"\t"`
	quotedValues         bool   `config:"QuotedValues"`
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
func (format *ProcessTSV) Configure(conf core.PluginConfigReader) {
	directives := conf.GetStringArray("Directives", []string{})

	format.directives = make([]tsvDirective, 0, len(directives))
	for _, d := range directives {
		directive := strings.Replace(d, "\\:", "\r", -1)
		parts := strings.Split(directive, ":")
		for i, value := range parts {
			parts[i] = strings.Replace(value, "\r", ":", -1)
		}

		if len(parts) >= 2 {
			index, err := strconv.Atoi(parts[0])
			conf.Errors.Push(err)

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
					format.Logger.Warning("ProcessTSV failed to parse a timestamp: ", err)
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
		remainder := string(format.GetAppliedContent(msg))
		for {
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
					format.Logger.Warning("ProcessTSV failed to parse a message: unmatched quote")
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
		values = stringsToTSVValues(strings.Split(string(format.GetAppliedContent(msg)), format.delimiter))
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

	format.SetAppliedContent(msg, []byte(strings.Join(stringValues, format.delimiter)))
	return nil // continue
}
