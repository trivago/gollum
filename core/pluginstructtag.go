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

package core

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// PluginStructTag extends reflect.StructTag by convenience methods used to
// retrieve auto configure values used by PluginConfigReader.
type PluginStructTag reflect.StructTag

const (
	// PluginStructTagDefault contains the tag name for default values
	PluginStructTagDefault = "default"
	// PluginStructTagMetric contains the tag name for metric values
	PluginStructTagMetric = "metric"
)

// GetBool returns the default boolean value for an auto configured field.
// When not set, false is returned.
func (tag PluginStructTag) GetBool() bool {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet || tagValue == "" {
		return false
	}
	value, err := strconv.ParseBool(tagValue)
	if err != nil {
		panic(err)
	}
	return value
}

// GetInt returns the default integer value for an auto configured field.
// When not set, 0 is returned. Metrics are not applied to this value.
func (tag PluginStructTag) GetInt() int64 {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet || tagValue == "" {
		return 0
	}

	var base int
	switch {
	case tagValue[0] == '0' && tagValue[1] == 'x':
		base = 16
	case tagValue[0] == '0':
		base = 8
	default:
		base = 10
	}

	value, err := strconv.ParseInt(tagValue, base, 64)
	if err != nil {
		panic(err)
	}
	return value
}

// GetUint returns the default unsigned integer value for an auto configured
// field. When not set, 0 is returned. Metrics are not applied to this value.
func (tag PluginStructTag) GetUint() uint64 {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet || tagValue == "" {
		return 0
	}

	var base int
	switch {
	case tagValue[0] == '0' && tagValue[1] == 'x':
		base = 16
	case tagValue[0] == '0':
		base = 8
	default:
		base = 10
	}

	value, err := strconv.ParseUint(tagValue, base, 64)
	if err != nil {
		panic(err)
	}
	return value
}

// GetString returns the default string value for an auto configured field.
// When not set, "" is returned.
func (tag PluginStructTag) GetString() string {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet {
		return ""
	}
	return tagValue
}

// GetStream returns the default message stream value for an auto configured
// field. When not set, InvalidStream is returned.
func (tag PluginStructTag) GetStream() MessageStreamID {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet {
		return InvalidStreamID
	}
	return GetStreamID(tagValue)
}

// GetStringArray returns the default string array value for an auto configured
// field. When not set an empty array is returned.
func (tag PluginStructTag) GetStringArray() []string {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet {
		return []string{}
	}
	return strings.Split(tagValue, ",")
}

// GetByteArray returns the default byte array value for an auto configured
// field. When not set an empty array is returned.
func (tag PluginStructTag) GetByteArray() []byte {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet {
		return []byte{}
	}
	return []byte(tagValue)
}

// GetStreamArray returns the default message stream array value for an auto
// configured field. When not set an empty array is returned.
func (tag PluginStructTag) GetStreamArray() []MessageStreamID {
	streamNames := tag.GetStringArray()
	streamIDs := make([]MessageStreamID, 0, len(streamNames))
	for _, name := range streamNames {
		streamIDs = append(streamIDs, GetStreamID(strings.TrimSpace(name)))
	}

	return streamIDs
}

// GetMetricScale returns the scale as defined by the "metric" tag as a number.
func (tag PluginStructTag) GetMetricScale() int64 {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagMetric)
	if !tagSet {
		return 1
	}

	switch strings.ToLower(tagValue) {
	case "b", "byte", "bytes":
		return 1
	case "kb", "kilobyte", "kilobytes":
		return 1 << 10
	case "mb", "megabyte", "megabytes":
		return 1 << 20
	case "gb", "gigabyte", "gigabytes":
		return 1 << 30
	case "tb", "terrabyte", "terrabytes":
		return 1 << 40

	case "ns", "nanosecond", "nanoseconds":
		return 1
	case "µs", "mcs", "microsecond", "microseconds":
		return int64(time.Microsecond)
	case "ms", "millisecond", "milliseconds":
		return int64(time.Millisecond)
	case "s", "sec", "second", "seconds":
		return int64(time.Second)
	case "m", "min", "minute", "minutes":
		return int64(time.Minute)
	case "h", "hour", "hours":
		return int64(time.Hour)
	case "d", "day", "days":
		return 24 * int64(time.Hour)
	case "w", "week", "weeks":
		return 7 * 24 * int64(time.Hour)

	default:
		return 1
	}
}
