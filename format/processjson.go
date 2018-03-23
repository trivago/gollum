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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mmcloughlin/geohash"
	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
	"gopkg.in/oschwald/geoip2-golang.v1"
)

// ProcessJSON formatter
//
// This formatter allows modification of JSON encoded data. Each field can be
// processed by different directives and the result of all directives will be
// stored back to the original location.
//
// Parameters
//
// - GeoIPFile: Defines a GeoIP file to load. This enables the "geoip"
// directive. If no file is loaded IPs will not be resolved. Files can be
// found e.g. at http://dev.maxmind.com/geoip/geoip2/geolite2/.
// By default this parameter is set to "".
//
// - TrimValues: Allows trimming of whitespaces from the beginning and end of
// each value after processing took place.
// By default this parameter is set to true.
//
// - Directives: Defines an array of actions to be applied to the JSON encoded
// data. Directives are processed in order of their appearance. Directives start
// with the name of the field, followed by an action followed by additional
// parameters if necessary. Parameters, key and action are separated by using
// the ":" character.
// By default this parameter is set to an empty list.
//
//  - split: <delimiter> {<key>, <key>, ...}
//  Split the field's value by the given delimiter, store the results to the
//  fields listed after the delimiter.
//
//  - replace: <string>  <new string>
//  Replace a given string inside the field's value with a new one.
//
//  - trim: <characters>
//  Remove the given characters from the start and end of the field's value.
//
//  - rename: <new key>
//  Rename a given field
//
//  - remove: {<value>, <value>, ...}`
//  Remove a given field. If additional parameters are given, the value is
//  expected to be an array. The given strings will be removed from that array.
//
//  - pick: <index> <key>
//  Pick a specific index from an array and store it to the given field.
//
//  - time: <from fromat> <to format>
//  Read a timestamp with a given format compatible to time.Parse and transform
//  it into another format compatible with time.Format.
//
//  - unixtimestamp: <unit> <to format>
//  Read a unix timestamp with a given unit ("s","ms" or "ns") and transform it
//  it into another format compatible with time.Format.
//
//  - flatten: {<delimiter>}
//  Move all keys from a nested object to new fields named
//  field + delimiter + subfield. If no delimiter is given "." will be used.
//
//  - agent: <prefix> {<field>, <field>, ...}
//  Parse the field's value as a user agent string and extract the given fields
//  into new fields named prefix + "_" + field.
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
//  - ip:
//  Parse the field as an array of strings and remove all values that cannot be
//  parsed as a valid IP. Single-string fields are supported, too, but will be
//  converted to an array.
//
//  - geoip: {<field>, <field>, ...}
//  Parse the field as an IP and extract the given fields into new fields named
//  prefix + "_" + field. This action requires a valid GeoIP file to be loaded.
//  If no fields are given all fields are returned.
//
//   - country: the contry code of the IP. Generates country, countryCode.
//   - city: the city of the IP
//   - continent: the continent of the IP. Generates continent, continentCode.
//   - timezone: the timezome of the IP
//   - proxy: name of the proxy if applying Generates proxy, satellite.
//   - location: the geolocation of this IP. Generates geocoord, geohash.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.ProcessJSON:
//        Directives:
//          - "host:split: :host:@timestamp"
//          - "@timestamp:time:20060102150405:2006-01-02 15\\:04\\:05"
//          - "client:ip"
//          - "client:geoip:location:country"
//          - "ua:agent:ua:os:engine:engine_version"
type ProcessJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	directives           []transformDirective
	trimValues           bool `config:"TrimValues" default:"true"`
	db                   *geoip2.Reader
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
func (format *ProcessJSON) Configure(conf core.PluginConfigReader) {
	directives := conf.GetStringArray("Directives", []string{})
	format.directives = make([]transformDirective, 0, len(directives))
	geoIPFile := conf.GetString("GeoIPFile", "")

	if geoIPFile != "" {
		var err error
		format.db, err = geoip2.Open(geoIPFile)
		conf.Errors.Push(err)
	}

	if len(directives) > 0 {
		format.directives = make([]transformDirective, 0, len(directives))
		for _, d := range directives {
			directive := strings.Replace(d, "\\:", "\r", -1)
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
				format.Logger.Warning(err.Error())
				return
			}

			if numParameters == 2 {

				if timestamp, err := time.Parse(directive.parameters[0], stringValue[:len(directive.parameters[0])]); err != nil {
					format.Logger.Warning("ProcessJSON failed to parse a timestamp: ", err)
				} else {
					(*values)[directive.key] = timestamp.Format(directive.parameters[1])
				}
			}

		case "unixtimestamp":
			floatValue, err := values.Float(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
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
				format.Logger.Warning(err.Error())
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

		case "pick":
			index, _ := strconv.Atoi(directive.parameters[0])
			field := directive.parameters[1]
			array, _ := (*values).Array(directive.key)

			if index < 0 {
				index = len(array) - index
			}
			if index < 0 || index >= len(array) {
				if len(array) > 0 {
					// Don't log if array is empty
					format.Logger.Warningf("Array index %d out of bounds: %#v", index, array)
				}
				return
			}
			(*values)[field] = array[index]

		case "remove":
			if numParameters == 0 {
				delete(*values, directive.key)
			} else {
				array, _ := (*values).Array(directive.key)
				wi := 0
			nextItem:
				for ri, value := range array {
					if strValue, isStr := value.(string); isStr {
						for _, cmp := range directive.parameters {
							if strValue == cmp {
								continue nextItem
							}
						}
					}
					if ri != wi {
						array[wi] = array[ri]
					}
					wi++
				}
				(*values)[directive.key] = array[:wi]
			}

		case "replace":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
				return
			}
			if numParameters == 2 {
				(*values)[directive.key] = strings.Replace(stringValue, tstrings.Unescape(directive.parameters[0]), tstrings.Unescape(directive.parameters[1]), -1)
			}

		case "trim":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
				return
			}
			switch {
			case numParameters == 0:
				(*values)[directive.key] = strings.Trim(stringValue, " \t")
			case numParameters == 1:
				(*values)[directive.key] = strings.Trim(stringValue, tstrings.Unescape(directive.parameters[0]))
			}

		case "flatten":
			delimiter := "."
			if numParameters == 1 {
				delimiter = directive.parameters[0]
			}
			keyPrefix := directive.key + delimiter
			if mapValue, err := values.MarshalMap(directive.key); err == nil {
				for key, val := range mapValue {
					(*values)[keyPrefix+key] = val
				}
			} else if arrayValue, err := values.Array(directive.key); err == nil {
				for index, val := range arrayValue {
					(*values)[keyPrefix+strconv.Itoa(index)] = val
				}
			} else {
				format.Logger.Warning("key was not a JSON array or object: " + directive.key)
				return
			}
			delete(*values, directive.key)

		case "agent":
			stringValue, err := values.String(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
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

		case "ip":
			ipArray, err := (*values).Array(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
				return
			}

			sanitized := make([]interface{}, 0, len(ipArray))
			for _, ip := range ipArray {
				ipString, isString := ip.(string)
				if !isString || net.ParseIP(ipString) != nil {
					sanitized = append(sanitized, ip)
				}
			}
			(*values)[directive.key] = sanitized

		case "geoip":
			if format.db == nil {
				return
			}

			ipString, err := (*values).String(directive.key)
			if err != nil {
				format.Logger.Warning(err.Error())
				return
			}

			fields := []string{
				"country",
				"city",
				"continent",
				"timezone",
				"proxy",
				"location",
			}
			if numParameters > 0 {
				fields = directive.parameters
			}

			ip := net.ParseIP(ipString)
			record, err := format.db.City(ip)
			if err != nil {
				format.Logger.Warningf("IP \"%s\" could not be resolved: %s", ipString, err.Error())
				format.Logger.Debugf("%#v", values)
				return
			}

			for _, field := range fields {
				switch field {
				case "city":
					name, exists := record.City.Names["en"]
					if exists {
						(*values)[directive.key+"_city"] = name
					}

				case "country":
					(*values)[directive.key+"_countryCode"] = record.Country.IsoCode
					name, exists := record.Country.Names["en"]
					if exists {
						(*values)[directive.key+"_country"] = name
					}

				case "continent":
					(*values)[directive.key+"_continentCode"] = record.Continent.Code
					name, exists := record.Continent.Names["en"]
					if exists {
						(*values)[directive.key+"_continent"] = name
					}

				case "timezone":
					(*values)[directive.key+"_timezone"] = record.Location.TimeZone

				case "proxy":
					(*values)[directive.key+"_proxy"] = record.Traits.IsAnonymousProxy
					(*values)[directive.key+"_satellite"] = record.Traits.IsSatelliteProvider

				case "location":
					(*values)[directive.key+"_geocoord"] = []float64{record.Location.Latitude, record.Location.Longitude}
					(*values)[directive.key+"_geohash"] = geohash.Encode(record.Location.Latitude, record.Location.Longitude)
				}
			}
		}
	}
}

// ApplyFormatter update message payload
func (format *ProcessJSON) ApplyFormatter(msg *core.Message) error {
	if len(format.directives) == 0 {
		return nil // ### return, no directives ###
	}

	values := make(tcontainer.MarshalMap)
	if err := json.Unmarshal(format.GetAppliedContent(msg), &values); err != nil {
		format.Logger.Warning("ProcessJSON failed to unmarshal a message: ", err)
		return err
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

	jsonData, err := json.Marshal(values)
	if err != nil {
		format.Logger.Warning("ProcessJSON failed to marshal a message: ", err)
		return err
	}

	format.SetAppliedContent(msg, jsonData)
	return nil
}
