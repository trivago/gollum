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

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	TypeName  string
	Enable    bool
	Channel   int
	Instances int
	Stream    []string
	Settings  ConfigKeyValueMap
}

// NewPluginConfig creates a new plugin config with default values.
// By default the plugin is enabled, has a buffered channel with 4096 slots, has
// one instance, is bound to no streams and has no additional settings.
func NewPluginConfig(pluginTypeName string) PluginConfig {
	return PluginConfig{
		TypeName:  pluginTypeName,
		Enable:    true,
		Channel:   4096,
		Instances: 1,
		Stream:    []string{},
		Settings:  make(ConfigKeyValueMap),
	}
}

// Read analyzes a given key/value map to extract the configuration values valid
// for each plugin. All non-default values are written to the Settings member.
func (conf *PluginConfig) Read(values ConfigKeyValueMap) {
	for key, settingValue := range values {
		switch key {
		case "Enable":
			conf.Enable = configReadBool("Enable", settingValue)

		case "Channel":
			conf.Channel = configReadInt("Channel", settingValue)

		case "Instances":
			conf.Instances = configReadInt("Instances", settingValue)

		case "Stream":
			switch settingValue.(type) {
			case string:
				conf.Stream = append(conf.Stream, settingValue.(string))
			default:
				arrayValue := configReadArray("Stream", settingValue)
				for _, value := range arrayValue {
					strValue := configReadString("An element of stream", value)
					conf.Stream = append(conf.Stream, strValue)
				}
			}

		default:
			conf.Settings[key] = settingValue
		}
	}

	// Sanity checks
	if conf.Instances == 0 {
		conf.Enable = false
	}
	if len(conf.Stream) == 0 {
		conf.Stream = append(conf.Stream, "*")
	}
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes non-predefined settings into account.
func (conf PluginConfig) HasValue(key string) bool {
	_, exists := conf.Settings[key]
	return exists
}

// Override sets or override a configuration value for non-predefined options.
func (conf PluginConfig) Override(key string, value interface{}) {
	conf.Settings[key] = value
}

// GetString tries to read a non-predefined, string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetString(key string, defaultValue string) string {
	value, exists := conf.Settings[key]
	if exists {
		return configReadString(key, value)
	}

	return defaultValue
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) []string {
	value, exists := conf.Settings[key]
	if exists {
		arrayValue := configReadStringArray(key, value)
		config := make([]string, 0, len(arrayValue))

		for _, value := range arrayValue {
			strValue := configReadString("An element of "+key, value)
			config = append(config, strValue)
		}
		return config
	}

	return defaultValue
}

// GetStringMap tries to read a non-predefined, string to string map from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringMap(key string, defaultValue map[string]string) map[string]string {
	mapping, exists := conf.Settings[key]
	if !exists {
		return defaultValue
	}

	result := make(map[string]string)
	mapValue := configReadMap(key, mapping)

	for keyItem, valItem := range mapValue {
		keyItemStr := configReadString("A key of "+key, keyItem)
		valItemStr := configReadString("A value of "+key, valItem)
		result[keyItemStr] = valItemStr
	}

	return result
}

// GetStreamMap tries to read a non-predefined, stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamMap(key string, defaultValue string) map[MessageStreamID]string {
	streamMap := make(map[MessageStreamID]string)
	streamMap[WildcardStreamID] = defaultValue

	mapping, exists := conf.Settings[key]
	if !exists {
		return streamMap
	}

	mapValue := configReadMap(key, mapping)

	for streamItem, targetItem := range mapValue {
		streamItemStr := configReadString("A key of "+key, streamItem)
		targetItemStr := configReadString("A value of "+key, targetItem)
		streamMap[GetStreamID(streamItemStr)] = targetItemStr
	}

	return streamMap
}

// GetStreamRoute tries to read a non-predefined, stream to stream map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamRoute(key string, defaultValue MessageStreamID) map[MessageStreamID][]MessageStreamID {
	streamRoute := make(map[MessageStreamID][]MessageStreamID)

	mapping, exists := conf.Settings[key]
	if !exists {
		streamRoute[WildcardStreamID] = []MessageStreamID{defaultValue}
		return streamRoute
	}

	mapValue := configReadMap(key, mapping)
	for sourceItem, targetItem := range mapValue {
		sourceItemStr := configReadString("A key of "+key, sourceItem)
		targetItemArray := configReadStringArray("A value of "+key, targetItem)
		sourceStream := GetStreamID(sourceItemStr)

		targetIds := []MessageStreamID{}
		for _, targetName := range targetItemArray {
			targetIds = append(targetIds, GetStreamID(targetName.(string)))
		}

		if _, exists := streamRoute[sourceStream]; exists {
			streamRoute[sourceStream] = append(streamRoute[sourceStream], targetIds...)
		} else {
			streamRoute[sourceStream] = targetIds
		}
	}

	if _, exists := streamRoute[WildcardStreamID]; !exists {
		streamRoute[WildcardStreamID] = []MessageStreamID{defaultValue}
	}

	return streamRoute
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) int {
	value, exists := conf.Settings[key]
	if exists {
		return configReadInt(key, value)
	}

	return defaultValue
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) bool {
	value, exists := conf.Settings[key]
	if exists {
		return configReadBool(key, value)
	}

	return defaultValue
}

// GetValue tries to read a non-predefined, untyped value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetValue(key string, defaultValue interface{}) interface{} {
	value, exists := conf.Settings[key]
	if exists {
		return value
	}

	return defaultValue
}
