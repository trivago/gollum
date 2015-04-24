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
	"github.com/trivago/gollum/shared"
)

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	TypeName  string
	Enable    bool
	Instances int
	Stream    []string
	Settings  shared.MarshalMap
}

// NewPluginConfig creates a new plugin config with default values.
// By default the plugin is enabled, has a buffered channel with 4096 slots, has
// one instance, is bound to no streams and has no additional settings.
func NewPluginConfig(pluginTypeName string) PluginConfig {
	return PluginConfig{
		TypeName:  pluginTypeName,
		Enable:    true,
		Instances: 1,
		Stream:    []string{},
		Settings:  shared.NewMarshalMap(),
	}
}

// Read analyzes a given key/value map to extract the configuration values valid
// for each plugin. All non-default values are written to the Settings member.
func (conf *PluginConfig) Read(values shared.MarshalMap) {
	var err error
	for key, settingValue := range values {
		switch key {
		case "Enable":
			conf.Enable, err = values.Bool("Enable")

		case "Instances":
			conf.Instances, err = values.Int("Instances")

		case "Stream":
			conf.Stream, err = values.StringArray("Stream")

		default:
			conf.Settings[key] = settingValue
		}
		if err != nil {
			Log.Error.Fatalf(err.Error())
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
	if conf.HasValue(key) {
		if value, err := conf.Settings.String(key); err != nil {
			Log.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}

	return defaultValue
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) []string {
	if conf.HasValue(key) {
		if value, err := conf.Settings.StringArray(key); err != nil {
			Log.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetStringMap tries to read a non-predefined, string to string map from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringMap(key string, defaultValue map[string]string) map[string]string {
	if conf.HasValue(key) {
		if value, err := conf.Settings.StringMap(key); err != nil {
			Log.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetStreamMap tries to read a non-predefined, stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamMap(key string, defaultValue string) map[MessageStreamID]string {
	streamMap := make(map[MessageStreamID]string)
	if defaultValue != "" {
		streamMap[WildcardStreamID] = defaultValue
	}
	if conf.HasValue(key) {
		if value, err := conf.Settings.StringMap(key); err != nil {
			Log.Error.Fatalf(err.Error())
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
	streamRoute := make(map[MessageStreamID][]MessageStreamID)

	if !conf.HasValue(key) {
		return streamRoute
	}

	if value, err := conf.Settings.StringArrayMap(key); err != nil {
		Log.Error.Fatalf(err.Error())
	} else {
		for sourceName, targets := range value {
			sourceStream := GetStreamID(sourceName)

			targetIds := []MessageStreamID{}
			for _, targetName := range targets {
				targetIds = append(targetIds, GetStreamID(targetName))
			}

			if _, exists := streamRoute[sourceStream]; exists {
				streamRoute[sourceStream] = append(streamRoute[sourceStream], targetIds...)
			} else {
				streamRoute[sourceStream] = targetIds
			}
		}
	}

	return streamRoute
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) int {
	if conf.HasValue(key) {
		if value, err := conf.Settings.Int(key); err != nil {
			Log.Error.Fatalf(err.Error())
		} else {
			return value
		}
	}
	return defaultValue
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) bool {
	if conf.HasValue(key) {
		if value, err := conf.Settings.Bool(key); err != nil {
			Log.Error.Fatalf(err.Error())
		} else {
			return value
		}
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
