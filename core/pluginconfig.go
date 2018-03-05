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
	"github.com/arbovm/levenshtein"
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"strings"
)

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	ID        string
	Typename  string
	Enable    bool
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
	if _, exists := conf.validKeys[key]; exists {
		return key // ### return, already registered ###
	}

	// Remove array notation from path
	path := key
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
		return key // ### return, already registered (without array notation) ###
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
	return key
}

func (conf PluginConfig) suggestKey(searchKey string) string {
	closestDist := len(searchKey)
	bestMatch := ""
	searchKeyLower := strings.ToLower(searchKey)

	for candidateKey := range conf.validKeys {
		candidateLower := strings.ToLower(candidateKey)
		dist := levenshtein.Distance(candidateLower, searchKeyLower)
		if dist == 0 {
			return candidateKey
		}
		if dist < closestDist {
			closestDist = dist
			bestMatch = candidateKey
		}
	}

	return bestMatch
}

// Validate should be called after a configuration has been processed. It will
// check the keys read from the config files against the keys requested up to
// this point. Unknown keys will be returned as errors
func (conf PluginConfig) Validate() error {
	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)
	for key := range conf.Settings {
		if _, exists := conf.validKeys[key]; !exists {
			if suggestion := conf.suggestKey(key); suggestion != "" {
				errors.Pushf("Unknown configuration key '%s' in '%s'. Did you mean '%s'?", key, conf.Typename, suggestion)
			} else {
				errors.Pushf("Unknown configuration key '%s' in '%s", key, conf.Typename)
			}
		}
	}
	return errors.OrNil()
}

func suggestType(typeName string) string {
	typeNameLower := strings.ToLower(typeName)
	closestDist := len(typeNameLower)
	bestMatch := ""

	allTypes := TypeRegistry.GetRegistered("")
	for _, candidateName := range allTypes {
		candidateLower := strings.ToLower(candidateName)
		dist := levenshtein.Distance(candidateLower, typeNameLower)
		if dist == 0 {
			return candidateName
		}
		if dist < closestDist {
			closestDist = dist
			bestMatch = candidateName
		}
	}

	return bestMatch
}

// Read analyzes a given key/value map to extract the configuration values valid
// for each plugin. All non-default values are written to the Settings member.
// All keys will be converted to lowercase.
func (conf *PluginConfig) Read(values tcontainer.MarshalMap) error {
	var err error
	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)
	for key, settingValue := range values {

		switch key {
		case "Type":
			conf.Typename, err = values.String(key)
			errors.Push(err)

		case "Enable":
			conf.Enable, err = values.Bool(key)
			errors.Push(err)

		default:
			// If the value is a marshalmap candidate -> convert it
			switch settingValue.(type) {
			case map[interface{}]interface{}, map[string]interface{}, tcontainer.MarshalMap:
				mmap, err := tcontainer.ConvertToMarshalMap(settingValue, nil)
				if !errors.Push(err) {
					conf.Settings[key] = mmap
				}

			default:
				conf.Settings[key] = settingValue
			}
		}
	}

	// Sanity checks and informal messages
	if conf.Typename == "" {
		errors.Pushf("Plugin '%s' does not define a type", conf.ID)
	}

	if !TypeRegistry.IsTypeRegistered(conf.Typename) {
		if suggestion := suggestType(conf.Typename); suggestion != "" {
			errors.Pushf("Plugin '%s' is using an unknown type '%s'. Did you mean '%s'?", conf.ID, conf.Typename, suggestion)
		} else {
			errors.Pushf("Plugin '%s' is using an unknown type '%s'.", conf.ID, conf.Typename)
		}
	}

	if !conf.Enable {
		logrus.Infof("Plugin %s has been disabled", conf.ID)
	}

	return errors.OrNil()
}

// Override sets or override a configuration value for non-predefined options.
func (conf *PluginConfig) Override(key string, value interface{}) {
	key = conf.registerKey(key)
	conf.Settings[key] = value
}
