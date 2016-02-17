// Copyright 2015-2016 trivago GmbH
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
)

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
	errorStack := tgo.NewErrorStack()
	return PluginConfigReader{
		WithError: NewPluginConfigReaderWithError(config),
		Errors:    &errorStack,
	}
}

// NewPluginConfigReaderFromReader encapsulates a WithError reader that
// is already attached to a config to read from.
func NewPluginConfigReaderFromReader(reader PluginConfigReaderWithError) PluginConfigReader {
	errorStack := tgo.NewErrorStack()
	return PluginConfigReader{
		WithError: reader,
		Errors:    &errorStack,
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

// GetLogScope creates a new tlog.LogScope for the plugin contained in this
// config.
func (reader *PluginConfigReader) GetLogScope() tlog.LogScope {
	return reader.WithError.GetLogScope()
}

// GetSubLogScope creates a new sub scope tlog.LogScope for the plugin contained
// in this config.
func (reader *PluginConfigReader) GetSubLogScope(subScope string) tlog.LogScope {
	return reader.WithError.GetSubLogScope(subScope)
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

// GetInt tries to read a integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (reader *PluginConfigReader) GetInt(key string, defaultValue int) int {
	value, err := reader.WithError.GetInt(key, defaultValue)
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
// to be passed and is overriden if the config specifies a type.
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
