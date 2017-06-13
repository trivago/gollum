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
	"github.com/mmcloughlin/geohash"
	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
	"gopkg.in/oschwald/geoip2-golang.v1"
	"net"
	"strconv"
	"strings"
	"time"
)

// ProcessJSON formatter plugin
// ProcessJSON is a formatter that allows modifications to fields of a given
// JSON message. The message is modified and returned again as JSON.
// Configuration example
//
//  - format.ProcessJSON:
//    GeoIPFile: ""
//    Directives:
//    - "host:split: :host:@timestamp"
//    - "@timestamp:time:20060102150405:2006-01-02 15\\:04\\:05"
//    - "error:replace:°:\n"
//    - "text:trim: \t"
//    - "foo:rename:bar"
//    - "foobar:remove"
//      - "array:pick:0:firstOfArray"
//    - "array:remove:foobar"
//      - "user_agent:agent:browser:os:version"
//      - "client:geoip:country:city:timezone:location"
//    TrimValues: true
//
// GeoIPFile defines a GeoIP file to load. This enables the "geoip"
// directive. If no file is loaded IPs will not be resolved. Files can be
// found e.g. at http://dev.maxmind.com/geoip/geoip2/geolite2/
//
// Directives defines the action to be applied to the json payload.
// Directives are processed in order of appearance.
// The directives have to be given in the form of key:operation:parameters, where
// operation can be one of the following.
// * `split:<string>{:<key>:<key>:...}` Split the value by a string and set the
//   resulting array elements to the given fields in order of appearance.
// * `replace:<old>:<new>` replace a given string in the value with a new one
// * `trim:<characters>` remove the given characters (not string!) from the start
//   and end of the value
// * `rename:<old>:<new>` rename a given field
// * `remove{:<string>:<string>...}` remove a given field. If additional parameters are
//   given, an array is expected. Strings given as additional parameters will be removed
//   from that array
// * `pick:<key>:<index>:<name>` Pick a specific index from an array and store it
//   in a new field.
// * `time:<read>:<write>` read a timestamp and transform it into another
//   format
// * `unixtimestamp:<read>:<write>` read a unix timestamp and transform it into another
//   format. valid read formats are s, ms, and ns.
// * `flatten{:<delimiter>}` create new fields from the values in field, with new
//   fields named field + delimiter + subfield. Delimiter defaults to ".".
//   Removes the original field.
// * `agent:<key>{:<field>:<field>:...}` Parse the value as a user agent string and
//   extract the given fields into <key>_<field>.
//   ("ua:agent:browser:os" would create the new fields "ua_browser" and "ua_os").
//   Possible values are: "mozilla", "platform", "os", "localization", "engine",
//   "engine_version", "browser", "version".
// * `ip` Parse the field as an array of strings and remove all values that cannot
//   be parsed as a valid IP. Single-string fields are supported, too, but will be
//   converted to an array.
// * `geoip:{<field>:<field>:...}` like agent this directive will analyse an IP string
//   via geoip and produce new fields.
//   Possible values are: "country", "city", "continent", "timezone", "proxy", "location"
//
// TrimValues will trim whitspaces from all values if enabled.
// Enabled by default.
type ProcessJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	directives           []transformDirective
	trimValues           bool
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
	format.trimValues = conf.GetBool("TrimValues", true)
	geoIPFile := conf.GetString("GeoIPFile", "")

	if geoIPFile != "" {
		var err error
		format.db, err = geoip2.Open(geoIPFile)
		conf.Errors.Push(err)
	}

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

		case "unixtimestamp":
			floatValue, err := values.Float(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
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
					format.Log.Warning.Printf("Array index %d out of bounds: %#v", index, array)
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
				format.Log.Warning.Print("key was not a JSON array or object: " + directive.key)
				return
			}
			delete(*values, directive.key)

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

		case "ip":
			ipArray, err := (*values).Array(directive.key)
			if err != nil {
				format.Log.Warning.Print(err.Error())
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
				format.Log.Warning.Print(err.Error())
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
				format.Log.Warning.Printf("IP \"%s\" could not be resolved: %s", ipString, err.Error())
				format.Log.Debug.Printf("%#v", values)
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
	err := json.Unmarshal(format.GetAppliedContent(msg), &values)
	if err != nil {
		format.Log.Warning.Print("ProcessJSON failed to unmarshal a message: ", err)
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
		format.Log.Warning.Print("ProcessJSON failed to marshal a message: ", err)
		return err
	}

	format.SetAppliedContent(msg, jsonData)
	return nil
}
