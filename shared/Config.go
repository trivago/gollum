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

package shared

import (
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	"reflect"
)

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	Enable   bool
	Channel  int
	Stream   []string
	Settings map[string]interface{}
}

// Config represents the top level config containing all plugin clonfigs
type Config struct {
	Settings map[string][]PluginConfig
}

func readBool(key string, val interface{}) bool {
	boolValue, isBool := val.(bool)
	if !isBool {
		log.Fatalf("Config parser: \"%s\" is expected to be a boolean.", key)
	}
	return boolValue
}

func readInt(key string, val interface{}) int {
	intValue, isInt := val.(int)
	if !isInt {
		log.Fatalf("Config parser: \"%s\" is expected to be an integer.", key)
	}
	return intValue
}

func readString(key string, val interface{}) string {
	strValue, isString := val.(string)
	if !isString {
		log.Fatalf("Config parser: \"%s\" is expected to be a string.", key)
	}
	return strValue
}

func readArray(key string, val interface{}) []interface{} {
	arrayValue, isArray := val.([]interface{})
	if !isArray {
		log.Fatalf("Config parser: \"%s\" is expected to be an array.", key)
	}
	return arrayValue
}

func readMap(key string, val interface{}) map[interface{}]interface{} {
	mapValue, isMap := val.(map[interface{}]interface{})
	if !isMap {
		log.Fatalf("Config parser: \"%s\" is expected to be a key/value map.", key)
	}
	return mapValue
}

// SetYAML is a YAMLReader interface implementation to convert values into the
// internal configuration format
func (conf Config) SetYAML(tagType string, values interface{}) bool {
	pluginList := values.([]interface{})
	stringType := reflect.TypeOf("")

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.

	for _, pluginData := range pluginList {
		pluginDataMap := pluginData.(map[interface{}]interface{})

		// Each item in the array item is a map{class -> map{key -> value}}
		// We "iterate" over the first map (one item only) to get the class.

		for pluginClass, pluginSettings := range pluginDataMap {
			pluginSettingsMap := pluginSettings.(map[interface{}]interface{})

			plugin := PluginConfig{
				Enable:   true,
				Channel:  4096,
				Stream:   []string{},
				Settings: make(map[string]interface{}),
			}

			// Iterate over all key/value pairs.
			// "Enable" is a special field as non-plugin logic is bound to it

			for settingKey, settingValue := range pluginSettingsMap {
				key := settingKey.(string)

				switch key {
				case "Enable":
					plugin.Enable = readBool("Enable", settingValue)

				case "Channel":
					plugin.Channel = readInt("Channel", settingValue)

				case "Stream":
					if reflect.TypeOf(settingValue) == stringType {
						plugin.Stream = append(plugin.Stream, settingValue.(string))
					} else {
						arrayValue := readArray("Stream", settingValue)
						for _, value := range arrayValue {
							strValue := readString("An element of stream", value)
							plugin.Stream = append(plugin.Stream, strValue)
						}
					}

				default:
					plugin.Settings[key] = settingValue
				}
			}

			// Set wildcard stream if no stream is set

			if len(plugin.Stream) == 0 {
				plugin.Stream = append(plugin.Stream, "*")
			}

			// Add instance of this plugin class config to the list

			list, listExists := conf.Settings[pluginClass.(string)]
			if !listExists {
				list = []PluginConfig{}
			}

			conf.Settings[pluginClass.(string)] = append(list, plugin)
		}
	}

	return true
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
		return readString(key, value)
	}

	return defaultValue
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) []string {
	value, exists := conf.Settings[key]
	if exists {
		strValue, isString := value.(string)
		if isString {
			return []string{strValue}
		}

		arrayValue := readArray(key, value)
		config := make([]string, 0, len(arrayValue))

		for _, value := range arrayValue {
			strValue := readString("An element of "+key, value)
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
	mapValue := readMap(key, mapping)

	for keyItem, valItem := range mapValue {
		keyItemStr := readString("A key of "+key, keyItem)
		valItemStr := readString("A value of "+key, valItem)
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

	mapValue := readMap(key, mapping)

	for streamItem, targetItem := range mapValue {
		streamItemStr := readString("A key of "+key, streamItem)
		targetItemStr := readString("A value of "+key, targetItem)
		streamMap[GetStreamID(streamItemStr)] = targetItemStr
	}

	return streamMap
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) int {
	value, exists := conf.Settings[key]
	if exists {
		return readInt(key, value)
	}

	return defaultValue
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) bool {
	value, exists := conf.Settings[key]
	if exists {
		return readBool(key, value)
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

// ReadConfig parses a YAML config file into a new Config struct.
func ReadConfig(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{make(map[string][]PluginConfig)}
	err = yaml.Unmarshal(buffer, conf)

	return conf, err
}
