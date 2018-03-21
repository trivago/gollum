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
	"fmt"
	"net/url"
	"reflect"
	"unsafe"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/treflect"
)

// Configurable defines an interface for structs that can be configured using
// a PluginConfigReader.
type Configurable interface {
	// Configure is called during NewPluginWithType
	Configure(PluginConfigReader)
}

// ScopedLogger defines an interface for structs that have a log scope attached
type ScopedLogger interface {
	GetLogger() logrus.FieldLogger
}

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
	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)
	return PluginConfigReader{
		WithError: NewPluginConfigReaderWithError(config),
		Errors:    &errors,
	}
}

// NewPluginConfigReaderFromReader encapsulates a WithError reader that
// is already attached to a config to read from.
func NewPluginConfigReaderFromReader(reader PluginConfigReaderWithError) PluginConfigReader {
	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)
	return PluginConfigReader{
		WithError: reader,
		Errors:    &errors,
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

// GetLogger creates a new scoped logger (logrus.FieldLogger) for the plugin
// contained in this config.
func (reader *PluginConfigReader) GetLogger() logrus.FieldLogger {
	return reader.WithError.GetLogger()
}

// GetSubLogger creates a new sub-scoped logger (logrus.FieldLogger) for the
// plugin contained in this config
func (reader *PluginConfigReader) GetSubLogger(subScope string) logrus.FieldLogger {
	return reader.WithError.GetSubLogger(subScope)
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

// GetURL tries to read an URL struct from a PluginConfig.
// If that value is not found nil is returned.
func (reader *PluginConfigReader) GetURL(key string, defaultValue string) *url.URL {
	value, err := reader.WithError.GetURL(key, defaultValue)
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
func (reader *PluginConfigReader) GetModulatorArray(key string, logger logrus.FieldLogger, defaultValue ModulatorArray) ModulatorArray {
	modulators, err := reader.WithError.GetModulatorArray(key, logger, defaultValue)
	reader.Errors.Push(err)
	return modulators
}

// GetFilterArray returns an array of filter plugins.
func (reader *PluginConfigReader) GetFilterArray(key string, logger logrus.FieldLogger, defaultValue FilterArray) FilterArray {
	filters, err := reader.WithError.GetFilterArray(key, logger, defaultValue)
	reader.Errors.Push(err)
	return filters
}

// GetFormatterArray returns an array of formatter plugins.
func (reader *PluginConfigReader) GetFormatterArray(key string, logger logrus.FieldLogger, defaultValue FormatterArray) FormatterArray {
	formatter, err := reader.WithError.GetFormatterArray(key, logger, defaultValue)
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

// Configure will configure a given item by scanning for plugin struct tags and
// calling the Configure method. Nested types will be traversed automatically.
func (reader *PluginConfigReader) Configure(item interface{}) error {
	itemValue := reflect.ValueOf(item)
	if itemValue.Kind() != reflect.Ptr {
		panic("Configure requires reference")
	}

	var logger logrus.FieldLogger
	if loggable, isScoped := item.(ScopedLogger); isScoped {
		logger = loggable.GetLogger()
	}

	reader.configureStruct(itemValue, logger)
	if confItem, isConfigurable := item.(Configurable); isConfigurable {
		confItem.Configure(*reader)
	}

	return reader.Errors.OrNil()
}

func (reader *PluginConfigReader) configureStruct(structVal reflect.Value, logger logrus.FieldLogger) {
	structType := treflect.RemovePtrFromType(structVal.Type())
	field := reflect.StructField{}

	// Reflection methods might panic. This hook provides context.
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("\"%s\": %v", field.Name, r))
		}
	}()

	for fieldIdx := 0; fieldIdx < structType.NumField(); fieldIdx++ {
		field = structType.Field(fieldIdx)
		fieldVal := treflect.RemovePtrFromValue(structVal).Field(fieldIdx)

		if key, hasFieldConfig := field.Tag.Lookup("config"); hasFieldConfig {
			reader.configureField(fieldVal, key, PluginStructTag(field.Tag), logger)
			continue // ### continue, configured by tag ###
		}

		fieldType := treflect.RemovePtrFromType(field.Type)
		if fieldType.Kind() != reflect.Struct {
			continue // ### continue, no nesting ###
		}

		if field.Type.Kind() == reflect.Ptr && fieldVal.IsNil() {
			continue // ### continue, ptr-to-nil struct ###
		}

		fieldValPtr := fieldVal.Addr()
		reader.configureStruct(fieldValPtr, logger)
		if !fieldValPtr.CanInterface() {
			continue // ### continue, cannot cast ###
		}

		if confItem, isConfigurable := fieldValPtr.Interface().(Configurable); isConfigurable {
			confItem.Configure(*reader)
		}
	}
}

func (reader *PluginConfigReader) configureField(fieldVal reflect.Value, key string, tags PluginStructTag, logger logrus.FieldLogger) {
	switch fieldVal.Kind() {
	case reflect.Bool:
		treflect.SetValue(fieldVal, reader.GetBool(key, tags.GetBool()))

	case reflect.String:
		treflect.SetValue(fieldVal, reader.GetString(key, tags.GetString()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		treflect.SetValue(fieldVal, reader.GetInt(key, tags.GetInt())*tags.GetMetricScale())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var value uint64
		if fieldVal.Type().Name() == "MessageStreamID" {
			value = uint64(reader.GetStreamID(key, tags.GetStream()))
		} else {
			scalar := uint64(tags.GetMetricScale())
			value = reader.GetUint(key, tags.GetUint()) * scalar
		}
		treflect.SetValue(fieldVal, value)

	case reflect.Array, reflect.Slice:
		reader.configureArrayField(fieldVal, key, tags, logger)

	case reflect.Interface:
		reader.configureInterfaceField(fieldVal, key, tags, logger)

	case reflect.Ptr:
		switch fieldVal.Type().Elem() {
		case reflect.TypeOf(url.URL{}):
			value := reader.GetURL(key, tags.GetString())
			treflect.SetValue(fieldVal, value)
		default:
			panic("Pointer field type not supported")
		}

	default:
		panic("Field type not supported")
	}
}

func (reader *PluginConfigReader) configureArrayField(fieldVal reflect.Value, key string, tags PluginStructTag,
	logger logrus.FieldLogger) {
	elementType := fieldVal.Type().Elem()

	switch elementType.Kind() {
	case reflect.String:
		treflect.SetValue(fieldVal, reader.GetStringArray(key, tags.GetStringArray()))

	case reflect.Int8, reflect.Uint8:
		treflect.SetValue(fieldVal, []byte(reader.GetString(key, tags.GetString())))

	case reflect.Uint64:
		if elementType.Name() == "MessageStreamID" {
			treflect.SetValue(fieldVal, reader.GetStreamArray(key, tags.GetStreamArray()))
		} else {
			panic("Field type not supported")
		}

	case reflect.Interface:
		switch elementType.Name() {
		case "Router":
			streams := reader.GetStreamArray(key, tags.GetStreamArray())
			routers := make([]Router, 0, len(streams))
			for _, streamID := range streams {
				if streamID != InvalidStreamID {
					routers = append(routers, StreamRegistry.GetRouterOrFallback(streamID))
				}
			}
			treflect.SetValue(fieldVal, routers)

		case "Modulator":
			modulators := reader.GetModulatorArray(key, logger, ModulatorArray{})
			treflect.SetValue(fieldVal, modulators)

		case "Filter":
			filters := reader.GetFilterArray(key, logger, FilterArray{})
			treflect.SetValue(fieldVal, filters)

		case "Formatter":
			formatters := reader.GetFormatterArray(key, logger, FormatterArray{})
			treflect.SetValue(fieldVal, formatters)
		}

	default:
		panic("Field type not supported")
	}

}

func (reader *PluginConfigReader) configureInterfaceField(fieldVal reflect.Value, key string, tags PluginStructTag,
	logger logrus.FieldLogger) {
	fieldType := treflect.RemovePtrFromType(fieldVal.Type())
	switch fieldType.Name() {
	case "Router":
		streamID := reader.GetStreamID(key, tags.GetStream())
		router := StreamRegistry.GetRouterOrFallback(streamID)

		if router != nil {
			// TODO: treflect.SetValue does not work here
			ptrToMember := unsafe.Pointer(fieldVal.UnsafeAddr())
			*(*Router)(ptrToMember) = router
		}

	default:
		panic("Field type not supported")
	}
}
