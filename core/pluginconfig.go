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
	Stream    []string
	Settings  tcontainer.MarshalMap
	validKeys map[string]bool
}

// NewPluginConfig creates a new plugin config with default values.
// By default the plugin is enabled, has one instance, is bound to no streams
// and has no additional settings.
func NewPluginConfig(pluginID string) PluginConfig {
	return PluginConfig{
		Enable:    true,
		Instances: 1,
		ID:        pluginID,
		Stream:    []string{},
		Settings:  tcontainer.NewMarshalMap(),
		validKeys: make(map[string]bool),
	}
}

// NewPluginConfigFromMap combines NewPluginConfig and the Read function
func NewPluginConfigFromMap(pluginID string, values tcontainer.MarshalMap) PluginConfig {
	conf := NewPluginConfig(pluginID)
	conf.Read(values)
	return conf
}

// registerKey registeres a key to the validKeys map as lowercase and returns
// the lowercase key
func (conf *PluginConfig) registerKey(key string) string {
	lowerCaseKey := strings.ToLower(key)
	conf.validKeys[lowerCaseKey] = true
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
func (conf *PluginConfig) Read(values tcontainer.MarshalMap) {
	var err error
	for key, settingValue := range values {
		lowerCaseKey := strings.ToLower(key)
		switch lowerCaseKey {
		case "type":
			conf.Typename, err = values.String("type")

		case "enable":
			conf.Enable, err = values.Bool("enable")

		case "instances":
			conf.Instances, err = values.Int("instances")

		case "stream":
			conf.Stream, err = values.StringArray("stream")

		default:
			conf.Settings[lowerCaseKey] = settingValue
		}

		if err != nil {
			tlog.Error.Fatalf("Plugin %s config contains an error: %s", conf.ID, err.Error())
		}
	}

	// Sanity checks and informal messages
	if conf.Typename == "" {
		tlog.Error.Fatalf("Plugin %s does not define a type", conf.ID)
	}

	if !TypeRegistry.IsTypeRegistered(conf.Typename) {
		tlog.Error.Fatalf("Plugin %s defines an unkown type %s", conf.ID, conf.Typename)
	}

	if conf.Instances == 0 {
		conf.Enable = false
		tlog.Warning.Printf("Plugin %s has been disabled (0 instances)", conf.ID)
	}

	if !conf.Enable {
		tlog.Note.Printf("Plugin %s has been disabled", conf.ID)
	}

	if len(conf.Stream) == 0 {
		conf.Stream = append(conf.Stream, conf.ID)
		tlog.Note.Printf("Plugin %s is using default stream name %s", conf.ID, conf.ID)
	}
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes non-predefined settings into account.
func (conf PluginConfig) HasValue(key string) bool {
	key = conf.registerKey(key)
	_, exists := conf.Settings[key]
	return exists
}

// Override sets or override a configuration value for non-predefined options.
func (conf PluginConfig) Override(key string, value interface{}) {
	key = conf.registerKey(key)
	conf.Settings[key] = value
}

// GetString tries to read a non-predefined, string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetString(key string, defaultValue string) string {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.String(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}

	return defaultValue
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) int {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.Int(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) bool {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.Bool(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
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

// GetArray tries to read a non-predefined, untyped array from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetArray(key string, defaultValue []interface{}) []interface{} {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.Array(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetMap tries to read a non-predefined, MarshalMap from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetMap(key string, defaultValue tcontainer.MarshalMap) tcontainer.MarshalMap {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.MarshalMap(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetPluginArray tries to read a non-predefined, array of plugins (type to config) from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetPluginArray(key string, defaultValue []Plugin) []Plugin {
	key = conf.registerKey(key)
	namedConfigs := conf.GetArray(key, []interface{}{})

	if len(namedConfigs) == 0 {
		return defaultValue
	}

	pluginArray := make([]Plugin, 0, len(namedConfigs))

	for _, typedConfigInterface := range namedConfigs {
		typedConfig, isMap := typedConfigInterface.(map[string]tcontainer.MarshalMap)
		if !isMap {
			tlog.Error.Fatalf("%s section is malformed", key)
		}

		for typeName, config := range typedConfig {
			pluginConfig := NewPluginConfigFromMap(typeName, config)
			plugin, err := NewPluginWithType(typeName, pluginConfig)
			if err != nil {
				tlog.Error.Fatalf("Plugin %s could not be instantiated: %s", typeName, err.Error())
			}
			pluginArray = append(pluginArray, plugin)
		}
	}

	return pluginArray
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) []string {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.StringArray(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetStringMap tries to read a non-predefined, string to string map from a
// PluginConfig. If the key is not found defaultValue is returned.
func (conf PluginConfig) GetStringMap(key string, defaultValue map[string]string) map[string]string {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		if value, err := conf.Settings.StringMap(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetStreamArray tries to read a non-predefined string array from a pluginconfig
// and translates all values to streamIds. If the key is not found defaultValue
// is returned.
func (conf PluginConfig) GetStreamArray(key string, defaultValue []MessageStreamID) []MessageStreamID {
	key = conf.registerKey(key)
	if conf.HasValue(key) {
		values := conf.GetStringArray(key, []string{})
		streamArray := []MessageStreamID{}
		for _, streamName := range values {
			streamArray = append(streamArray, GetStreamID(streamName))
		}
		return streamArray
	}

	return defaultValue
}

// GetStreamMap tries to read a non-predefined, stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamMap(key string, defaultValue string) map[MessageStreamID]string {
	key = conf.registerKey(key)
	streamMap := make(map[MessageStreamID]string)
	if defaultValue != "" {
		streamMap[WildcardStreamID] = defaultValue
	}

	if conf.HasValue(key) {
		if value, err := conf.Settings.StringMap(key); err != nil {
			tlog.Error.Fatalf(err.Error())
		} else {
			for streamName, target := range value {
				streamMap[GetStreamID(streamName)] = target
			}
		}
	}

	return streamMap
}

// GetStreamRoutes tries to read a non-predefined, stream to stream map from a
// plugin config. If no routes are defined an empty map is returned
func (conf PluginConfig) GetStreamRoutes(key string) map[MessageStreamID][]MessageStreamID {
	key = conf.registerKey(key)
	streamRoute := make(map[MessageStreamID][]MessageStreamID)
	if !conf.HasValue(key) {
		return streamRoute
	}

	if value, err := conf.Settings.StringArrayMap(key); err != nil {
		tlog.Error.Fatalf(err.Error())
	} else {
		for sourceName, targets := range value {
			sourceStream := GetStreamID(sourceName)

			targetIds := []MessageStreamID{}
			for _, targetName := range targets {
				targetIds = append(targetIds, GetStreamID(targetName))
			}

			streamRoute[sourceStream] = targetIds
		}
	}

	return streamRoute
}
