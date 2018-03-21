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
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tstrings"
	"net/url"
)

// PluginConfigReaderWithError is a read-only wrapper on top of a plugin config
// that provides convenience functions for accessing values from the
// wrapped config.
type PluginConfigReaderWithError struct {
	config *PluginConfig
}

// NewPluginConfigReaderWithError creates a new raw reader for a given config.
// In contrast to the PluginConfigReaderWithError all functions return an error
// next to the value.
func NewPluginConfigReaderWithError(config *PluginConfig) PluginConfigReaderWithError {
	return PluginConfigReaderWithError{
		config: config,
	}
}

// GetLogger creates a logger scoped for the plugin contained in this config.
func (reader PluginConfigReaderWithError) GetLogger() logrus.FieldLogger {
	return logrus.WithFields(logrus.Fields{
		"PluginType": reader.config.Typename,
		"PluginID":   reader.config.ID,
	})
}

// GetSubLogger creates a logger scoped gor the plugin contained in this config,
// with an additional subscope.
func (reader PluginConfigReaderWithError) GetSubLogger(subScope string) logrus.FieldLogger {
	return reader.GetLogger().WithField("Scope", subScope)
}

// GetID returns the plugin's id
func (reader PluginConfigReaderWithError) GetID() string {
	return reader.config.ID
}

// GetTypename returns the plugin's typename
func (reader PluginConfigReaderWithError) GetTypename() string {
	return reader.config.Typename
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes settings into account.
func (reader PluginConfigReaderWithError) HasValue(key string) bool {
	key = reader.config.registerKey(key)
	_, exists := reader.config.Settings.Value(key)
	return exists
}

// GetString tries to read a string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetString(key string, defaultValue string) (string, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		value, err := reader.config.Settings.String(key)
		return tstrings.Unescape(value), err
	}
	return defaultValue, nil
}

// GetURL tries to read an URL struct from a PluginConfig.
// If that value is not found nil is returned.
func (reader PluginConfigReaderWithError) GetURL(key string, defaultValue string) (*url.URL, error) {
	value, err := reader.GetString(key, defaultValue)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	urlValue, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	return urlValue, nil
}

// GetInt tries to read a integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetInt(key string, defaultValue int64) (int64, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		if strVal, err := reader.config.Settings.String(key); err == nil {
			return tstrings.AtoI64(strVal) // Allow string to number conversion
		}
		return reader.config.Settings.Int(key)
	}
	return defaultValue, nil
}

// GetUint tries to read an unsigned integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetUint(key string, defaultValue uint64) (uint64, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		if strVal, err := reader.config.Settings.String(key); err == nil {
			return tstrings.AtoU64(strVal) // Allow string to number conversion
		}
		return reader.config.Settings.Uint(key)
	}
	return defaultValue, nil
}

// GetFloat tries to read a float value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetFloat(key string, defaultValue float64) (float64, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.Float(key)
	}
	return defaultValue, nil
}

// GetBool tries to read a boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetBool(key string, defaultValue bool) (bool, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.Bool(key)
	}
	return defaultValue, nil
}

// GetValue tries to read a untyped value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetValue(key string, defaultValue interface{}) interface{} {
	key = reader.config.registerKey(key)
	value, exists := reader.config.Settings[key]
	if exists {
		return value
	}
	return defaultValue
}

// GetStreamID tries to read a StreamID value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetStreamID(key string, defaultValue MessageStreamID) (MessageStreamID, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		value, err := reader.config.Settings.String(key)
		if err != nil {
			return InvalidStreamID, err
		}
		return GetStreamID(value), nil
	}
	return defaultValue, nil
}

// GetPlugin creates a nested plugin from a config map. The default type has
// to be passed and is overridden if the config specifies a type.
// The value stored in the config can either be a string or a map. If a map
// is found it is used to override defaultConfig. If a string is found it is
// used to override defaultType.
func (reader PluginConfigReaderWithError) GetPlugin(key string, defaultType string, defaultConfig tcontainer.MarshalMap) (Plugin, error) {
	key = reader.config.registerKey(key)
	configMap := defaultConfig

	if reader.HasValue(key) {
		if value, err := reader.config.Settings.String(key); err == nil {
			defaultType = value
		} else {
			configMap, err = reader.GetMap(key, defaultConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	config, err := NewNestedPluginConfig(defaultType, configMap)
	if err != nil {
		return nil, err
	}
	return NewPluginWithConfig(config)
}

// GetArray tries to read a untyped array from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetArray(key string, defaultValue []interface{}) ([]interface{}, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.Array(key)
	}
	return defaultValue, nil
}

// GetMap tries to read a MarshalMap from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetMap(key string, defaultValue tcontainer.MarshalMap) (tcontainer.MarshalMap, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.MarshalMap(key)
	}
	return defaultValue, nil
}

// GetPluginArray tries to read a array of plugins (type to config) from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetPluginArray(key string, defaultValue []Plugin) ([]Plugin, error) {
	key = reader.config.registerKey(key)
	namedConfigs, err := reader.GetArray(key, []interface{}{})
	if err != nil {
		return defaultValue, err
	}

	if len(namedConfigs) == 0 {
		return defaultValue, nil
	}

	pluginArray := make([]Plugin, 0, len(namedConfigs))

	// Iterate over all entries in the array.
	// An entry can either be a string or a string -> map[string]interface{}
	for _, typedConfigInterface := range namedConfigs {
		typedConfigMap := tcontainer.TryConvertToMarshalMap(typedConfigInterface, nil)
		typedConfig, isMap := typedConfigMap.(tcontainer.MarshalMap)

		if !isMap {
			typeNameStr, isString := typedConfigInterface.(string)
			if !isString {
				return nil, fmt.Errorf("%s section is malformed (entry does not contain a config)", key)
			}

			// Only string given, initialize with empty config
			pluginConfig, _ := NewNestedPluginConfig(typeNameStr, tcontainer.NewMarshalMap())
			plugin, err := NewPluginWithConfig(pluginConfig)
			if err != nil {
				return nil, err
			}
			pluginArray = append(pluginArray, plugin)
		}

		// string -> map[string]interface{}
		// This is actually a map with just one entry
		for typename, config := range typedConfig {
			configMap, err := tcontainer.ConvertToMarshalMap(config, nil)
			if err != nil {
				return nil, fmt.Errorf("%s section is malformed (config for %s is not a map but %T)", key, typename, config)
			}

			pluginConfig, _ := NewNestedPluginConfig(typename, configMap)
			plugin, err := NewPluginWithConfig(pluginConfig)
			if err != nil {
				return pluginArray, err
			}
			pluginArray = append(pluginArray, plugin)
		}
	}

	return pluginArray, nil
}

// GetModulatorArray returns an array of modulator plugins.
func (reader PluginConfigReaderWithError) GetModulatorArray(key string, logger logrus.FieldLogger,
	defaultValue ModulatorArray) (ModulatorArray, error) {
	modulators := []Modulator{}

	modPlugins, err := reader.GetPluginArray(key, []Plugin{})
	if err != nil {
		return modulators, err
	}
	if len(modPlugins) == 0 {
		return modulators, nil
	}

	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)

	for _, plugin := range modPlugins {
		if filter, isFilter := plugin.(Filter); isFilter {
			filterModulator := NewFilterModulator(filter)
			modulators = append(modulators, filterModulator)
		} else if formatter, isFormatter := plugin.(Formatter); isFormatter {
			formatterModulator := NewFormatterModulator(formatter)
			modulators = append(modulators, formatterModulator)
		} else if modulator, isModulator := plugin.(Modulator); isModulator {
			if modulator, isScopedModulator := plugin.(ScopedModulator); isScopedModulator {
				modulator.SetLogger(logger)
			}
			modulators = append(modulators, modulator)
		} else {
			errors.Pushf("Plugin '%T' is not a valid modulator", plugin)
			panic(errors.Top())
		}
	}

	return modulators, errors.OrNil()
}

// GetFilterArray returns an array of filter plugins.
func (reader PluginConfigReaderWithError) GetFilterArray(key string, logger logrus.FieldLogger,
	defaultValue FilterArray) (FilterArray, error) {
	filters := []Filter{}

	modPlugins, err := reader.GetPluginArray(key, []Plugin{})
	if err != nil {
		return filters, err
	}
	if len(modPlugins) == 0 {
		return filters, nil
	}

	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)

	for _, plugin := range modPlugins {
		if filter, isFilter := plugin.(Filter); isFilter {
			filters = append(filters, filter)
		} else {
			errors.Pushf("Plugin '%T' is not a valid filter", plugin)
			panic(errors.Top())
		}
	}

	return filters, errors.OrNil()
}

// GetFormatterArray returns an array of formatter plugins.
func (reader PluginConfigReaderWithError) GetFormatterArray(key string, logger logrus.FieldLogger, defaultValue FormatterArray) (FormatterArray, error) {
	formatters := []Formatter{}

	modPlugins, err := reader.GetPluginArray(key, []Plugin{})
	if err != nil {
		return formatters, err
	}
	if len(modPlugins) == 0 {
		return formatters, nil
	}

	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)

	for _, plugin := range modPlugins {
		if formatter, isFormatter := plugin.(Formatter); isFormatter {
			formatters = append(formatters, formatter)
		} else {
			errors.Pushf("Plugin '%T' is not a valid formatter", plugin)
			panic(errors.Top())
		}
	}

	return formatters, errors.OrNil()
}

// GetStringArray tries to read a string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetStringArray(key string, defaultValue []string) ([]string, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.StringArray(key)
	}
	return defaultValue, nil
}

// GetStringMap tries to read a string to string map from a
// PluginConfig. If the key is not found defaultValue is returned.
func (reader PluginConfigReaderWithError) GetStringMap(key string, defaultValue map[string]string) (map[string]string, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		return reader.config.Settings.StringMap(key)
	}
	return defaultValue, nil
}

// GetStreamArray tries to read a string array from a pluginconfig
// and translates all values to streamIds. If the key is not found defaultValue
// is returned.
func (reader PluginConfigReaderWithError) GetStreamArray(key string, defaultValue []MessageStreamID) ([]MessageStreamID, error) {
	key = reader.config.registerKey(key)
	if reader.HasValue(key) {
		values, err := reader.GetStringArray(key, []string{})
		if err != nil {
			return nil, err
		}
		streamArray := []MessageStreamID{}
		for _, streamName := range values {
			streamArray = append(streamArray, GetStreamID(streamName))
		}
		return streamArray, nil
	}

	return defaultValue, nil
}

// GetStreamMap tries to read a stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (reader PluginConfigReaderWithError) GetStreamMap(key string, defaultValue string) (map[MessageStreamID]string, error) {
	key = reader.config.registerKey(key)
	streamMap := make(map[MessageStreamID]string)
	if defaultValue != "" {
		streamMap[WildcardStreamID] = defaultValue
	}

	if reader.HasValue(key) {
		value, err := reader.config.Settings.StringMap(key)
		if err != nil {
			return nil, err
		}

		for streamName, target := range value {
			streamMap[GetStreamID(streamName)] = target
		}
	}

	return streamMap, nil
}

// GetStreamRoutes tries to read a stream to stream map from a
// plugin config. If no routes are defined an empty map is returned
func (reader PluginConfigReaderWithError) GetStreamRoutes(key string, defaultValue map[MessageStreamID][]MessageStreamID) (map[MessageStreamID][]MessageStreamID, error) {
	key = reader.config.registerKey(key)
	if !reader.HasValue(key) {
		return defaultValue, nil
	}

	streamRoute := make(map[MessageStreamID][]MessageStreamID)
	value, err := reader.config.Settings.StringArrayMap(key)
	if err != nil {
		return nil, err
	}

	for sourceName, targets := range value {
		sourceStream := GetStreamID(sourceName)
		targetIds := []MessageStreamID{}
		for _, targetName := range targets {
			targetIds = append(targetIds, GetStreamID(targetName))
		}

		streamRoute[sourceStream] = targetIds
	}

	return streamRoute, nil
}
