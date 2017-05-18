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

// registerKey registers a key to the validKeys map as lowercase and returns
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
	var err error
	errors := tgo.NewErrorStack()
	for key, settingValue := range values {
		lowerCaseKey := strings.ToLower(key)

		switch lowerCaseKey {
		case "type":
			conf.Typename, err = values.String(key)
			errors.Push(err)

		case "enable":
			conf.Enable, err = values.Bool(key)
			errors.Push(err)

		case "instances":
			conf.Instances, err = values.Int(key)
			errors.Push(err)

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

// Override sets or override a configuration value for non-predefined options.
func (conf *PluginConfig) Override(key string, value interface{}) {
	key = conf.registerKey(key)
	conf.Settings[key] = value
}
