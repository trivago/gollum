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
	"strconv"
	"strings"
	"time"
)

// JSONToInflux10 formatter plugin
// JSONToInflux10 provides a transformation from JSON data to
// InfluxDB 0.9.1+ compatible line protocol data.
// Configuration example
//
//  - format.JSONToInflux10
//      TimeField: time
//      Measurement: measurement
//      Ignore:
//        - ignore
//        - these
//        - tags
//      Tags:
//        - datacenter
//        - service
//        - host
//        - application
//
// TimeField specifies the JSON field that holds the Unix timestamp of the message.
// The precision is seconds.
// By default, the field is named `time`. If such a key does not exist in the
// message, the current Unix timestamp at the time of parsing the message is used.
//
// Measurement specifies the JSON field that holds the measurement of the message.
// By default, the field is named `measurement`. This field MUST exist in the message.
// If it does not exist in the message, an error will be thrown.
//
// Ignore lists all JSON fields that will be ignored and not sent to InfluxDB.
//
// Tags lists all names of JSON fields to send to Influxdb as tags.
// The Influxdb 0.9 convention is that values that do not change every
// request should be considered metadata and given as tags.
//
// JsonToInfluxFormatter defines the formatter applied before the conversion
// from JSON to InfluxDB. By default this is set to format.Forward.
type JSONToInflux10 struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	metadataEscape       *strings.Replacer
	measurementEscape    *strings.Replacer
	fieldValueEscape     *strings.Replacer
	timeField            string `config:"Timefield" default:"time"`
	measurement          string `config:"Measurement" default:"measurement"`
	tags                 map[string]bool
	ignore               map[string]bool
}

func init() {
	core.TypeRegistry.Register(JSONToInflux10{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSONToInflux10) Configure(conf core.PluginConfigReader) {
	format.SimpleFormatter.Configure(conf)

	format.metadataEscape = strings.NewReplacer(",", "\\,", "=", "\\=", " ", "\\ ")
	format.measurementEscape = strings.NewReplacer(",", "\\,", " ", "\\ ")
	format.fieldValueEscape = strings.NewReplacer("\"", "\\\"")

	// Create a set from the tags array for O(1) lookup
	tagsArray := conf.GetStringArray("Tags", []string{})
	format.tags = map[string]bool{}
	for _, v := range tagsArray {
		format.tags[v] = true
	}

	// Create a set from the ignore array for O(1) lookup
	ignoreArray := conf.GetStringArray("Ignore", []string{})
	format.ignore = map[string]bool{}
	for _, v := range ignoreArray {
		format.ignore[v] = true
	}
}

func (format *JSONToInflux10) escapeMeasurement(value string) string {
	return format.measurementEscape.Replace(value)
}

func (format *JSONToInflux10) escapeMetadata(value string) string {
	return format.metadataEscape.Replace(value)
}

func (format *JSONToInflux10) escapeFieldValue(value string) string {
	return format.fieldValueEscape.Replace(value)
}

func (format *JSONToInflux10) joinMap(m map[string]interface{}) string {
	var tokens []string
	for k, v := range m {
		tokens = append(tokens, fmt.Sprintf(`%s=%s`, k, v))
	}
	return fmt.Sprintf(strings.Join(tokens, ","))
}

// ApplyFormatter updates the message payload
func (format *JSONToInflux10) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)

	values := make(tcontainer.MarshalMap)
	err := json.Unmarshal(content, &values)

	if err != nil {
		format.Logger.Warningf("JSON parser error: %s, Message: %s", err, content)
		return err
	}

	var timestamp int64
	if val, ok := values[format.timeField]; ok {
		timestamp, err = strconv.ParseInt(val.(string), 10, 64)
		if err != nil {
			format.Logger.Error("Invalid time format in message:", err)
		}
		delete(values, format.timeField)
	} else {
		timestamp = time.Now().Unix()
	}

	measurement, measurementFound := values[format.measurement]
	if !measurementFound {
		return fmt.Errorf("Required field for measurement (%s) not found in payload", format.measurement)
	}

	delete(values, format.measurement)

	fields := make(map[string]interface{})
	tags := make(map[string]interface{})
	for k, v := range values {
		if _, ignore := format.ignore[k]; ignore {
			continue
		}
		key := format.escapeMetadata(k)
		if _, isTag := format.tags[k]; isTag {
			tags[key] = format.escapeMetadata(v.(string))
		} else {
			fields[key] = format.escapeFieldValue(v.(string))
		}
	}

	line := fmt.Sprintf(
		`%s,%s %s %d`,
		format.escapeMeasurement(measurement.(string)),
		format.joinMap(tags),
		format.joinMap(fields),
		timestamp)

	format.SetAppliedContent(msg, []byte(line))
	return nil
}
