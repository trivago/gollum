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

package core

import (
	"github.com/trivago/tgo/tstrings"
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
	if !tagSet || len(tagValue) == 0 {
		return 0
	}

	value, err := tstrings.AtoI64(tagValue)
	if err != nil {
		panic(err)
	}
	return value
}

// GetUint returns the default unsigned integer value for an auto configured
// field. When not set, 0 is returned. Metrics are not applied to this value.
func (tag PluginStructTag) GetUint() uint64 {
	tagValue, tagSet := reflect.StructTag(tag).Lookup(PluginStructTagDefault)
	if !tagSet || len(tagValue) == 0 {
		return 0
	}

	value, err := tstrings.AtoU64(tagValue)
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
	return tstrings.Unescape(tagValue)
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

	return metricScale[strings.ToLower(tagValue)]
}

const (
	metricScaleB   = int64(1)
	metricScaleKB  = int64(1) << 10
	metricScaleMB  = int64(1) << 20
	metricScaleGB  = int64(1) << 30
	metricScaleTB  = int64(1) << 40
	metricScaleNs  = int64(time.Nanosecond)
	metricScaleMcs = int64(time.Microsecond)
	metricScaleMs  = int64(time.Millisecond)
	metricScaleS   = int64(time.Second)
	metricScaleM   = int64(time.Minute)
	metricScaleH   = int64(time.Hour)
	metricScaleD   = 24 * int64(time.Hour)
	metricScaleW   = 7 * 24 * int64(time.Hour)
)

var metricScale = map[string]int64{
	"b":     metricScaleB,
	"byte":  metricScaleB,
	"bytes": metricScaleB,

	"kb":        metricScaleKB,
	"kilobyte":  metricScaleKB,
	"kilobytes": metricScaleKB,

	"mb":        metricScaleMB,
	"megabyte":  metricScaleMB,
	"megabytes": metricScaleMB,

	"gb":        metricScaleGB,
	"gigabyte":  metricScaleGB,
	"gigabytes": metricScaleGB,

	"tb":        metricScaleTB,
	"terabyte":  metricScaleTB,
	"terabytes": metricScaleTB,

	"ns":          metricScaleNs,
	"nanosecond":  metricScaleNs,
	"nanoseconds": metricScaleNs,

	"µs":           metricScaleMcs,
	"mcs":          metricScaleMcs,
	"microsecond":  metricScaleMcs,
	"microseconds": metricScaleMcs,

	"ms":           metricScaleMs,
	"millisecond":  metricScaleMs,
	"milliseconds": metricScaleMs,

	"s":       metricScaleS,
	"sec":     metricScaleS,
	"second":  metricScaleS,
	"seconds": metricScaleS,

	"m":       metricScaleM,
	"min":     metricScaleM,
	"minute":  metricScaleM,
	"minutes": metricScaleM,

	"h":     metricScaleH,
	"hour":  metricScaleH,
	"hours": metricScaleH,

	"d":    metricScaleD,
	"day":  metricScaleD,
	"days": metricScaleD,

	"w":     metricScaleW,
	"week":  metricScaleW,
	"weeks": metricScaleW,
}
