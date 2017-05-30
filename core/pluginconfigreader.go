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

package core

import (
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tlog"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// PluginConfigReader provides another convenience wrapper on top of
// a PluginConfigReaderWithError. All functions of this reader are stripped
// from returning errors. Errors from calls are collected internally
// and can be accessed at any given point.
type PluginConfigReader struct {
	WithError PluginConfigReaderWithError
	Errors    *tgo.ErrorStack
}

// NewPluginConfigReader creates a new reader on top of a given config.
func NewPluginConfigReader(config *PluginConfig) PluginConfigReader {
	errorStack := tgo.NewErrorStack()
	return PluginConfigReader{
		WithError: NewPluginConfigReaderWithError(config),
		Errors:    &errorStack,
	}
}

// NewPluginConfigReaderFromReader encapsulates a WithError reader that
// is already attached to a config to read from.
func NewPluginConfigReaderFromReader(reader PluginConfigReaderWithError) PluginConfigReader {
	errorStack := tgo.NewErrorStack()
	return PluginConfigReader{
		WithError: reader,
		Errors:    &errorStack,
	}
}

// GetID returns the plugin's id
func (reader *PluginConfigReader) GetID() string {
	return reader.WithError.GetID()
}

// GetTypename returns the plugin's typename
func (reader *PluginConfigReader) GetTypename() string {
	return reader.WithError.GetTypename()
}

// GetLogScope creates a new tlog.LogScope for the plugin contained in this
// config.
func (reader *PluginConfigReader) GetLogScope() tlog.LogScope {
	return reader.WithError.GetLogScope()
}

// GetSubLogScope creates a new sub scope tlog.LogScope for the plugin contained
// in this config.
func (reader *PluginConfigReader) GetSubLogScope(subScope string) tlog.LogScope {
	return reader.WithError.GetSubLogScope(subScope)
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes settings into account.
func (reader *PluginConfigReader) HasValue(key string) bool {
	return reader.WithError.HasValue(key)
}

// GetString tries to read a string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetString(key string, defaultValue string) string {
	value, err := reader.WithError.GetString(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetInt tries to read a integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetInt(key string, defaultValue int64) int64 {
	value, err := reader.WithError.GetInt(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetUint tries to read an unsigned integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetUint(key string, defaultValue uint64) uint64 {
	value, err := reader.WithError.GetUint(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetBool tries to read a boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetBool(key string, defaultValue bool) bool {
	value, err := reader.WithError.GetBool(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetValue tries to read a untyped value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetValue(key string, defaultValue interface{}) interface{} {
	return reader.WithError.GetValue(key, defaultValue)
}

// GetStreamID tries to read a StreamID value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetStreamID(key string, defaultValue MessageStreamID) MessageStreamID {
	value, err := reader.WithError.GetStreamID(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetPlugin creates a nested plugin from a config map. The default type has
// to be passed and is overridden if the config specifies a type.
// The value stored in the config can either be a string or a map. If a map
// is found it is used to override defaultConfig. If a string is found it is
// used to override defaultType.
func (reader *PluginConfigReader) GetPlugin(key string, defaultType string, defaultConfig tcontainer.MarshalMap) Plugin {
	value, err := reader.WithError.GetPlugin(key, defaultType, defaultConfig)
	reader.Errors.Push(err)
	return value
}

// GetArray tries to read a untyped array from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetArray(key string, defaultValue []interface{}) []interface{} {
	value, err := reader.WithError.GetArray(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetMap tries to read a MarshalMap from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetMap(key string, defaultValue tcontainer.MarshalMap) tcontainer.MarshalMap {
	value, err := reader.WithError.GetMap(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetPluginArray tries to read a array of plugins (type to config) from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetPluginArray(key string, defaultValue []Plugin) []Plugin {
	value, err := reader.WithError.GetPluginArray(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetModulatorArray reads an array of modulator plugins
func (reader *PluginConfigReader) GetModulatorArray(key string, logScope tlog.LogScope, defaultValue ModulatorArray) ModulatorArray {
	modulators, err := reader.WithError.GetModulatorArray(key, logScope, defaultValue)
	reader.Errors.Push(err)
	return modulators
}

// GetFilterArray returns an array of filter plugins.
func (reader *PluginConfigReader) GetFilterArray(key string, logScope tlog.LogScope, defaultValue FilterArray) FilterArray {
	filters, err := reader.WithError.GetFilterArray(key, logScope, defaultValue)
	reader.Errors.Push(err)
	return filters
}

// GetFormatterArray returns an array of formatter plugins.
func (reader *PluginConfigReader) GetFormatterArray(key string, logScope tlog.LogScope, defaultValue FormatterArray) FormatterArray {
	formatter, err := reader.WithError.GetFormatterArray(key, logScope, defaultValue)
	reader.Errors.Push(err)
	return formatter
}

// GetStringArray tries to read a string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetStringArray(key string, defaultValue []string) []string {
	value, err := reader.WithError.GetStringArray(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetStringMap tries to read a string to string map from a
// PluginConfig. If the key is not found defaultValue is returned.
func (reader *PluginConfigReader) GetStringMap(key string, defaultValue map[string]string) map[string]string {
	value, err := reader.WithError.GetStringMap(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetStreamArray tries to read a string array from a pluginconfig
// and translates all values to streamIds. If the key is not found defaultValue
// is returned.
func (reader *PluginConfigReader) GetStreamArray(key string, defaultValue []MessageStreamID) []MessageStreamID {
	value, err := reader.WithError.GetStreamArray(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetStreamMap tries to read a stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (reader *PluginConfigReader) GetStreamMap(key string, defaultValue string) map[MessageStreamID]string {
	value, err := reader.WithError.GetStreamMap(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// GetStreamRoutes tries to read a stream to stream map from a
// plugin config. If no routes are defined an empty map is returned
func (reader *PluginConfigReader) GetStreamRoutes(key string, defaultValue map[MessageStreamID][]MessageStreamID) map[MessageStreamID][]MessageStreamID {
	value, err := reader.WithError.GetStreamRoutes(key, defaultValue)
	reader.Errors.Push(err)
	return value
}

// Configure reads struct tags from the given plugin and sets the tagged values
// according to config and/or defaults.
// Avaiable tags: "config" holds the config key, "default" holds the default
// value, "metric" may store metric quantity information like "kb" or "sec".
func (reader *PluginConfigReader) Configure(plugin interface{}, log tlog.LogScope) {
	pluginType := reflect.TypeOf(plugin).Elem()
	pluginValue := reflect.ValueOf(plugin).Elem()
	numFields := pluginType.NumField()

	for i := 0; i < numFields; i++ {
		field := pluginType.Field(i)
		fielValue := pluginValue.Field(i)

		if key, haskey := field.Tag.Lookup("config"); haskey {
			defaultTag, _ := field.Tag.Lookup("default")
			metric, _ := field.Tag.Lookup("metric")

			switch field.Type.Kind() {
			case reflect.Bool:
				if defaultTag == "" {
					defaultTag = "false"
				}
				defaultValue, err := strconv.ParseBool(defaultTag)
				reader.Errors.Push(err)
				fielValue.SetBool(reader.GetBool(key, defaultValue))

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if defaultTag == "" {
					defaultTag = "0"
				}
				base := 10
				if len(defaultTag) > 1 && defaultTag[0] == '0' {
					base = 8
				}
				defaultValue, err := strconv.ParseInt(defaultTag, base, 64)
				reader.Errors.Push(err)
				value := metricScale(metric) * reader.GetInt(key, defaultValue)
				fielValue.SetInt(value)

			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var value uint64
				if field.Type.Name() == "MessageStreamID" {
					value = uint64(GetStreamID(reader.GetString(key, defaultTag)))
				} else {
					if defaultTag == "" {
						defaultTag = "0"
					}
					base := 10
					if len(defaultTag) > 1 && defaultTag[0] == '0' {
						base = 8
					}
					defaultValue, err := strconv.ParseUint(defaultTag, base, 64)
					reader.Errors.Push(err)
					value = uint64(metricScale(metric)) * reader.GetUint(key, defaultValue)
				}
				fielValue.SetUint(value)

			case reflect.String:
				fielValue.SetString(reader.GetString(key, defaultTag))

			case reflect.Array, reflect.Slice:
				reader.readArray(key, fielValue, field.Type.Elem(), log, defaultTag)

			case reflect.Interface:
				reader.readInterface(key, fielValue, field.Type, log, defaultTag)
			}
		}
	}
}

func (reader *PluginConfigReader) readArray(key string, pluginValue reflect.Value, arrayType reflect.Type, log tlog.LogScope, defaultTag string) {
	elements := strings.Split(defaultTag, ",")
	switch arrayType.Kind() {
	case reflect.String:
		defaultValue := make([]string, 0, len(elements))
		for _, e := range elements {
			defaultValue = append(defaultValue, e)
		}
		arrayValue := reflect.ValueOf(reader.GetStringArray(key, defaultValue))
		pluginValue.Set(arrayValue)

	case reflect.Int8, reflect.Uint8:
		defaultValue := reader.GetString(key, defaultTag)
		byteArray := []byte(defaultValue)
		pluginValue.Set(reflect.ValueOf(byteArray))

	case reflect.Uint64:
		if arrayType.Name() == "MessageStreamID" {
			defaultValue := make([]MessageStreamID, 0, len(elements))
			for _, e := range elements {
				defaultValue = append(defaultValue, GetStreamID(e))
			}
			arrayValue := reflect.ValueOf(reader.GetStreamArray(key, defaultValue))
			pluginValue.Set(arrayValue)
		}

	case reflect.Interface:
		switch arrayType.Name() {
		case "Router":
			defaultStreams := []MessageStreamID{GetStreamID(defaultTag)}
			streams := reader.GetStreamArray(key, defaultStreams)
			router := make([]Router, 0, len(streams))
			for _, streamID := range streams {
				if streamID != InvalidStreamID {
					router = append(router, StreamRegistry.GetRouterOrFallback(streamID))
				}
			}
			pluginValue.Set(reflect.ValueOf(router))

		case "Modulator":
			modulators := reader.GetModulatorArray(key, log, ModulatorArray{})
			pluginValue.Set(reflect.ValueOf(modulators))

		case "Filter":
			filters := reader.GetFilterArray(key, log, FilterArray{})
			pluginValue.Set(reflect.ValueOf(filters))

		case "Formatter":
			formatters := reader.GetFormatterArray(key, log, FormatterArray{})
			pluginValue.Set(reflect.ValueOf(formatters))
		}
	}
}

func (reader *PluginConfigReader) readInterface(key string, pluginValue reflect.Value, fieldType reflect.Type, log tlog.LogScope, defaultTag string) {
	switch fieldType.Name() {
	case "Router":
		streamID := reader.GetStreamID(key, GetStreamID(defaultTag))
		if streamID != InvalidStreamID {
			router := StreamRegistry.GetRouterOrFallback(streamID)
			pluginValue.Set(reflect.ValueOf(router))
		}
	}
}

func metricScale(metric string) int64 {
	switch strings.ToLower(metric) {
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
