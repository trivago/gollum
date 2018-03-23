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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// JSONToInflux10 formatter
//
// JSONToInflux10 provides a transformation from arbitrary JSON data to
// InfluxDB 0.9.1+ compatible line protocol data.
//
// Parameters
//
// - TimeField: Specifies the JSON field that holds the timestamp of the message.
// The timestamp is formatted as defined by TimeFormat. If the field is not
// found the current timestamp is assumed.
// By default this parameter is set to "time".
//
// - TimeFormat: Specifies the format of the time field as in go's time.Parse
// or "unix" if the field contains a valid unix timestamp.
// By default this parameter is set to "unix".
//
// - Measurement: Specifies the JSON field that holds the measurements in this
// message. If the field doesn't exist, the message is discarded.
// By default this parameter is set to "measurement".
//
// - Ignore: May contain a list of JSON fields that should be ignored and not
// sent to InfluxDB.
// By default this parameter is set to an empty list.
//
// - Tags: May contain a list of JSON fields to send to InfluxDB as tags.
// The InfluxDB 0.9 convention is that values that do not change by every
// request are to be considered metadata and given as tags.
//
// Examples
//
//  metricsToInflux:
//    Type: producer.InfluxDB
//    Streams: metrics
//    Host: "influx01:8086"
//    Database: "metrics"
//    Modulators:
//      - format.JSONToInflux10
//        TimeField: timestamp
//        Measurement: metrics
//        Tags:
//          - tags
//          - service
//          - host
//          - application
type JSONToInflux10 struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	metadataEscape       *strings.Replacer
	measurementEscape    *strings.Replacer
	fieldValueEscape     *strings.Replacer
	timeField            string `config:"TimeField" default:"time"`
	timeFormat           string `config:"TimeFormat" default:"unix"`
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
	return fmt.Sprint(strings.Join(tokens, ","))
}

// Return the time field as a unix timestamp
func (format *JSONToInflux10) toUnixTime(timeString string) (int64, error) {
	if format.timeFormat == "unix" {
		return strconv.ParseInt(timeString, 10, 64)
	}
	t, err := time.Parse(format.timeFormat, timeString)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
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
		timestamp, err = format.toUnixTime(val.(string))
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

	measurementString := format.escapeMeasurement(measurement.(string))

	tagsString := format.joinMap(tags)
	if len(tagsString) > 0 {
		// Only add comma between measurement and tags if tags are not empty.
		// See https://docs.influxdata.com/influxdb/v1.2/write_protocols/line_protocol_reference/
		tagsString = "," + tagsString
	}

	fieldsString := format.joinMap(fields)

	line := fmt.Sprintf(
		`%s%s %s %d`,
		measurementString,
		tagsString,
		fieldsString,
		timestamp)

	format.SetAppliedContent(msg, []byte(line))
	return nil
}
