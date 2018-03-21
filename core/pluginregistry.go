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
	"sync"
)

type pluginRegistry struct {
	plugins map[string]Plugin
	guard   *sync.RWMutex
}

// PluginRegistry holds all plugins by their name
var PluginRegistry = pluginRegistry{
	plugins: make(map[string]Plugin),
	guard:   new(sync.RWMutex),
}

// RegisterUnique stores a plugin by its ID (a string) for later retrieval.
// Name collisions are not resolved, duplicated names will not be registered.
func (registry *pluginRegistry) RegisterUnique(plugin Plugin, ID string) bool {
	registry.guard.Lock()
	defer registry.guard.Unlock()

	if _, exists := registry.plugins[ID]; !exists {
		registry.plugins[ID] = plugin
		return true
	}

	return false
}

// GetPlugin returns a plugin by name or nil if not found.
func (registry *pluginRegistry) GetPlugin(ID string) Plugin {
	registry.guard.RLock()
	plugin, exists := registry.plugins[ID]
	registry.guard.RUnlock()

	if exists {
		return plugin
	}
	return nil
}

// GetPluginWithState returns a plugin by name if it has a state or nil.
func (registry *pluginRegistry) GetPluginWithState(ID string) PluginWithState {
	plugin := registry.GetPlugin(ID)
	if pluginWithState, hasState := plugin.(PluginWithState); hasState {
		return pluginWithState
	}
	return nil
}
