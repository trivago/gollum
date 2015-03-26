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
	"github.com/trivago/gollum/core/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// ConfigKeyValueMap is used by PluginConfig to store setting->value mappings
type ConfigKeyValueMap map[string]interface{}

// Config represents the top level config containing all plugin clonfigs
type Config struct {
	Values  []map[string]ConfigKeyValueMap
	Plugins []PluginConfig
}

// ReadConfig parses a YAML config file into a new Config struct.
func ReadConfig(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	err = yaml.Unmarshal(buffer, &config.Values)
	if err != nil {
		return nil, err
	}

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.
	for _, pluginData := range config.Values {

		// Each item in the array item is a map{class -> map{key -> value}}
		// We "iterate" over the first map (one item only) to get the class.
		for typeName, pluginSettings := range pluginData {
			plugin := NewPluginConfig(typeName)
			plugin.Read(pluginSettings)
			config.Plugins = append(config.Plugins, plugin)
		}
	}

	return config, err
}

func configReadBool(key string, val interface{}) bool {
	boolValue, isBool := val.(bool)
	if !isBool {
		Log.Error.Fatalf("Parser: \"%s\" is expected to be a boolean.", key)
	}
	return boolValue
}

func configReadInt(key string, val interface{}) int {
	intValue, isInt := val.(int)
	if !isInt {
		Log.Error.Fatalf("Parser: \"%s\" is expected to be an integer.", key)
	}
	return intValue
}

func configReadString(key string, val interface{}) string {
	strValue, isString := val.(string)
	if !isString {
		Log.Error.Fatalf("Parser: \"%s\" is expected to be a string.", key)
	}
	return strValue
}

func configReadStringArray(key string, val interface{}) []interface{} {
	switch val.(type) {
	case string:
		return []interface{}{configReadString(key, val)}
	default:
		return configReadArray(key, val)
	}
}

func configReadArray(key string, val interface{}) []interface{} {
	arrayValue, isArray := val.([]interface{})
	if !isArray {
		Log.Error.Fatalf("Parser: \"%s\" is expected to be an array.", key)
	}
	return arrayValue
}

func configReadMap(key string, val interface{}) map[interface{}]interface{} {
	mapValue, isMap := val.(map[interface{}]interface{})
	if !isMap {
		Log.Error.Fatalf("Parser: \"%s\" is expected to be a key/value map.", key)
	}
	return mapValue
}
