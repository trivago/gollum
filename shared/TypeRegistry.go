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
	"reflect"
	"strings"
)

// typeRegistryError is returned by typeRegistry functions.
type typeRegistryError struct {
	message string
}

// Error interface implementation for typeRegistryError.
func (err typeRegistryError) Error() string {
	return err.message
}

// typeRegistry is a name to type registry used to create objects by name.
type typeRegistry struct {
	namedType map[string]reflect.Type
}

// Plugin is the global typeRegistry singleton.
// Use this singleton to register plugins.
var RuntimeType = typeRegistry{make(map[string]reflect.Type)}

// Register a plugin to the typeRegistry by passing an uninitialized object.
// Example: var MyConsumerClassID = shared.Plugin.Register(MyConsumer{})
func (registry typeRegistry) Register(typeInstance interface{}) {
	structType := reflect.TypeOf(typeInstance)
	pathIdx := strings.LastIndex(structType.PkgPath(), "/") + 1

	typeName := structType.PkgPath()[pathIdx:] + "." + structType.Name()
	registry.namedType[typeName] = structType
}

// New creates an uninitialized object by class name.
// The class name has to be "package.class" or "package/subpackage.class".
// The gollum package is omitted from the package path.
func (registry typeRegistry) New(typeName string) (interface{}, error) {
	structType, exists := registry.namedType[typeName]
	if exists {
		return reflect.New(structType).Interface(), nil
	}
	return nil, typeRegistryError{"Unknown class: " + typeName}
}

// NewPluginWithType creates a new plugin of a given type and initializes it
// using the given config (i.e. passes that config to Configure). The type
// passed to this function may differ from the type stored in the config.
// If the type is meant to match use NewPlugin instead of NewPluginWithType.
// This function returns nil, error if the plugin could not be instantiated or
// plugin, error if Configure failed.
func (registry typeRegistry) NewPluginWithType(typeName string, config PluginConfig) (Plugin, error) {
	obj, err := registry.New(typeName)
	if err != nil {
		return nil, err
	}

	plugin, isPlugin := obj.(Plugin)
	if !isPlugin {
		return nil, typeRegistryError{typeName + " is no plugin."}
	}

	err = plugin.Configure(config)
	return plugin, err
}

// NewPlugin creates a new plugin from the type information stored in its
// config. This function internally calls NewPluginWithType.
func (registry typeRegistry) NewPlugin(config PluginConfig) (Plugin, error) {
	return registry.NewPluginWithType(config.TypeName, config)
}

// GetRegistered returns the names of all registered types for a given package
func (registry typeRegistry) GetRegistered(packageName string) []string {
	var result []string
	for key := range registry.namedType {
		if strings.HasPrefix(key, packageName) {
			result = append(result, key)
		}
	}
	return result
}
