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

// NewPlugin creates a new plugin and initializes it with the given config.
// If the plugin failed to configure the plugin and an error is returned.
// If another error occured plugin will be nil.
func (registry typeRegistry) NewPlugin(typeName string, config PluginConfig) (Plugin, error) {
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
