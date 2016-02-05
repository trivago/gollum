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
	"fmt"
)

type pluginRegistry struct {
	plugins map[string]Plugin
}

// PluginRegistry holds all plugins by their name
var PluginRegistry = pluginRegistry{
	plugins: make(map[string]Plugin),
}

// Register stores a plugin by its ID (a string) for later retrieval.
// Name collisions are resolved automatically by adding an incrementing number
// to the name. As of this the registered name is returned by this function.
func (registry *pluginRegistry) Register(plugin Plugin, ID string) string {
	collision := 1
	pluginID := ID
	for {
		if _, exists := registry.plugins[pluginID]; !exists {
			registry.plugins[pluginID] = plugin
			return pluginID // ### return, registered ###
		}
		pluginID = fmt.Sprintf("%s%d", ID, collision)
		collision++
	}
}

// RegisterUnique stores a plugin by its ID (a string) for later retrieval.
// Name collisions are not resolved, duplicated names will not be registered.
func (registry *pluginRegistry) RegisterUnique(plugin Plugin, ID string) {
	if _, exists := registry.plugins[ID]; !exists {
		registry.plugins[ID] = plugin
	}
}

// GetPlugin returns a plugin by name or nil if not found.
func (registry *pluginRegistry) GetPlugin(ID string) Plugin {
	plugin, exists := registry.plugins[ID]
	if !exists {
		plugin = nil
	}
	return plugin
}

// GetPluginWithState returns a plugin by name if it has a state or nil.
func (registry *pluginRegistry) GetPluginWithState(ID string) PluginWithState {
	plugin := registry.GetPlugin(ID)
	pluginWithState, hasState := plugin.(PluginWithState)
	if !hasState {
		pluginWithState = nil
	}
	return pluginWithState
}
