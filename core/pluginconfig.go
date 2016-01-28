// Copyright 2015 trivago GmbH
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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tlog"
	"strings"
)

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	ID        string
	Typename  string
	Enable    bool
	Instances int
	Settings  tcontainer.MarshalMap
	validKeys map[string]bool
}

// NewPluginConfig creates a new plugin config with default values.
// By default the plugin is enabled, has one instance and has no additional settings.
// Passing an empty pluginID makes the plugin anonymous.
// The defaultTypename may be overridden by a later call to read.
func NewPluginConfig(pluginID string, defaultTypename string) PluginConfig {
	return PluginConfig{
		Enable:    true,
		Instances: 1,
		ID:        pluginID,
		Typename:  defaultTypename,
		Settings:  tcontainer.NewMarshalMap(),
		validKeys: make(map[string]bool),
	}
}

// NewNestedPluginConfig creates a pluginconfig based on a given set of config
// values. The plugin created does not have an id, i.e. it is considered
// anonymous.
func NewNestedPluginConfig(defaultTypename string, values tcontainer.MarshalMap) (PluginConfig, error) {
	conf := NewPluginConfig("", defaultTypename)
	err := conf.Read(values)
	return conf, err
}

// registerKey registeres a key to the validKeys map as lowercase and returns
// the lowercase key
func (conf *PluginConfig) registerKey(key string) string {
	lowerCaseKey := strings.ToLower(key)
	if _, exists := conf.validKeys[lowerCaseKey]; exists {
		return lowerCaseKey // ### return, already registered ###
	}

	// Remove array notation from path
	path := lowerCaseKey
	startIdx := strings.IndexRune(path, tcontainer.MarshalMapArrayBegin)
	for startIdx > -1 {
		if endIdx := strings.IndexRune(path[startIdx:], tcontainer.MarshalMapArrayEnd); endIdx > -1 {
			path = path[0:startIdx] + path[startIdx+endIdx+1:]
			startIdx = strings.IndexRune(path, tcontainer.MarshalMapArrayBegin)
		} else {
			startIdx = -1
		}
	}

	if _, exists := conf.validKeys[path]; exists {
		return lowerCaseKey // ### return, already registered (without array notation) ###
	}

	// Register all parts of the path
	startIdx = strings.IndexRune(path, tcontainer.MarshalMapSeparator)
	cutIdx := startIdx
	for startIdx > -1 {
		conf.validKeys[path[:cutIdx]] = true
		startIdx = strings.IndexRune(path[startIdx+1:], tcontainer.MarshalMapSeparator)
		cutIdx += startIdx
	}

	conf.validKeys[path] = true
	return lowerCaseKey
}

// Validate should be called after a configuration has been processed. It will
// check the keys read from the config files against the keys requested up to
// this point. Unknown keys will be written to the error log.
func (conf PluginConfig) Validate() bool {
	valid := true
	for key := range conf.Settings {
		if _, exists := conf.validKeys[key]; !exists {
			valid = false
			tlog.Warning.Printf("Unknown configuration key in %s: %s", conf.Typename, key)
		}
	}
	return valid
}

// Read analyzes a given key/value map to extract the configuration values valid
// for each plugin. All non-default values are written to the Settings member.
// All keys will be converted to lowercase.
func (conf *PluginConfig) Read(values tcontainer.MarshalMap) error {
	errors := tgo.NewErrorStack()
	for key, settingValue := range values {
		lowerCaseKey := strings.ToLower(key)

		switch lowerCaseKey {
		case "type":
			conf.Typename = errors.String(values.String(key))

		case "enable":
			conf.Enable = errors.Bool(values.Bool(key))

		case "instances":
			conf.Instances = errors.Int(values.Int(key))

		default:
			// If the value is a marshalmap candidate -> convert it
			switch settingValue.(type) {
			case map[interface{}]interface{}, map[string]interface{}, tcontainer.MarshalMap:
				mmap, err := tcontainer.ConvertToMarshalMap(settingValue, strings.ToLower)
				if !errors.Push(err) {
					conf.Settings[lowerCaseKey] = mmap
				}

			default:
				conf.Settings[lowerCaseKey] = settingValue
			}
		}
	}

	// Sanity checks and informal messages
	if conf.Typename == "" {
		errors.Pushf("Plugin %s does not define a type", conf.ID)
	}

	if !TypeRegistry.IsTypeRegistered(conf.Typename) {
		errors.Pushf("Plugin %s is using an unkown type %s", conf.ID, conf.Typename)
	}

	if conf.Instances <= 0 {
		conf.Enable = false
		tlog.Warning.Printf("Plugin %s has been disabled (0 instances)", conf.ID)
	}

	if !conf.Enable {
		tlog.Note.Printf("Plugin %s has been disabled", conf.ID)
	}

	return errors.OrNil()
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes non-predefined settings into account.
func (conf PluginConfig) HasValue(key string) bool {
	key = conf.registerKey(key)
	_, exists := conf.Settings.Value(key)
	return exists
}

// Override sets or override a configuration value for non-predefined options.
func (conf PluginConfig) Override(key string, value interface{}) {
	key = conf.registerKey(key)
	conf.Settings[key] = value
}

// GetString tries to read a non-predefined, string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetString(key string, defaultValue string) (string, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.String(key)
	}
	return defaultValue, nil
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) (int, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.Int(key)
	}
	return defaultValue, nil
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) (bool, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.Bool(key)
	}
	return defaultValue, nil
}

// GetValue tries to read a non-predefined, untyped value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetValue(key string, defaultValue interface{}) interface{} {
	key = conf.registerKey(key)
	value, exists := conf.Settings[key]
	if exists {
		return value
	}
	return defaultValue
}

// GetStreamID tries to read a non-predefined, StreamID value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStreamID(key string, defaultValue MessageStreamID) (MessageStreamID, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		value, err := conf.Settings.String(key)
		if err != nil {
			return InvalidStreamID, err
		}
		return GetStreamID(value), nil
	}
	return defaultValue, nil
}

// GetPlugin creates a nested plugin from a config map. The default type has
// to be passed and is overriden if the config specifies a type.
// The value stored in the config can either be a string or a map. If a map
// is found it is used to override defaultConfig. If a string is found it is
// used to override defaultType.
func (conf PluginConfig) GetPlugin(key string, defaultType string, defaultConfig tcontainer.MarshalMap) (Plugin, error) {
	key = conf.registerKey(key)
	configMap := defaultConfig

	if conf.HasValue(key) {
		if value, err := conf.Settings.String(key); err == nil {
			defaultType = value
		} else {
			configMap, err = conf.GetMap(key, defaultConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	config, err := NewNestedPluginConfig(defaultType, configMap)
	if err != nil {
		return nil, err
	}
	return NewPlugin(config)
}

// GetArray tries to read a non-predefined, untyped array from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetArray(key string, defaultValue []interface{}) ([]interface{}, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.Array(key)
	}
	return defaultValue, nil
}

// GetMap tries to read a non-predefined, MarshalMap from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetMap(key string, defaultValue tcontainer.MarshalMap) (tcontainer.MarshalMap, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.MarshalMap(key)
	}
	return defaultValue, nil
}

// GetPluginArray tries to read a non-predefined, array of plugins (type to config) from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetPluginArray(key string, defaultValue []Plugin) ([]Plugin, error) {
	key = conf.registerKey(key)
	namedConfigs, err := conf.GetArray(key, []interface{}{})
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
		typedConfig, isMap := typedConfigInterface.(map[interface{}]interface{})
		if !isMap {
			typeNameStr, isString := typedConfigInterface.(string)
			if !isString {
				return nil, fmt.Errorf("%s section is malformed (entry does not contain a config)", key)
			}

			// Only string given, initialize with empty config
			pluginConfig, _ := NewNestedPluginConfig(typeNameStr, tcontainer.NewMarshalMap())
			plugin, err := NewPlugin(pluginConfig)
			if err != nil {
				return nil, err
			}
			pluginArray = append(pluginArray, plugin)
		}

		// string -> map[string]interface{}
		// This is actually a map with just one entry
		for typename, config := range typedConfig {
			typeNameStr, isString := typename.(string)
			if !isString {
				return nil, fmt.Errorf("%s section is malformed (entry is not type:config)", key)
			}

			configMap, err := tcontainer.ConvertToMarshalMap(config, nil)
			if err != nil {
				return nil, fmt.Errorf("%s section is malformed (config for %s is not a map but %T)", key, typeNameStr, config)
			}

			pluginConfig, err := NewNestedPluginConfig(typeNameStr, configMap)
			plugin, err := NewPlugin(pluginConfig)
			if err != nil {
				return pluginArray, err
			}
			pluginArray = append(pluginArray, plugin)
		}
	}

	return pluginArray, nil
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) ([]string, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.StringArray(key)
	}
	return defaultValue, nil
}

// GetStringMap tries to read a non-predefined, string to string map from a
// PluginConfig. If the key is not found defaultValue is returned.
func (conf PluginConfig) GetStringMap(key string, defaultValue map[string]string) (map[string]string, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		return conf.Settings.StringMap(key)
	}
	return defaultValue, nil
}

// GetStreamArray tries to read a non-predefined string array from a pluginconfig
// and translates all values to streamIds. If the key is not found defaultValue
// is returned.
func (conf PluginConfig) GetStreamArray(key string, defaultValue []MessageStreamID) ([]MessageStreamID, error) {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		values, err := conf.GetStringArray(key, []string{})
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

// GetStreamMap tries to read a non-predefined, stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamMap(key string, defaultValue string) (map[MessageStreamID]string, error) {
	key = conf.registerKey(key)
	streamMap := make(map[MessageStreamID]string)
	if defaultValue != "" {
		streamMap[WildcardStreamID] = defaultValue
	}

	if conf.HasValue(key) {
		value, err := conf.Settings.StringMap(key)
		if err != nil {
			return nil, err
		}

		for streamName, target := range value {
			streamMap[GetStreamID(streamName)] = target
		}
	}

	return streamMap, nil
}

// GetStreamRoutes tries to read a non-predefined, stream to stream map from a
// plugin config. If no routes are defined an empty map is returned
func (conf PluginConfig) GetStreamRoutes(key string) (map[MessageStreamID][]MessageStreamID, error) {
	key = conf.registerKey(key)
	streamRoute := make(map[MessageStreamID][]MessageStreamID)
	if !conf.HasValue(key) {
		return streamRoute, nil
	}

	value, err := conf.Settings.StringArrayMap(key)
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
