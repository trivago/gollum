package shared

import (
	"reflect"
	"strings"
)

const (
	gollumPackageName = "gollum/"
)

// ClassRegistryError is returned by ClassRegistry functions.
type ClassRegistryError struct {
	Message string
}

// Error interface implementation for ClassRegistryError.
func (err ClassRegistryError) Error() string {
	return err.Message
}

// ClassRegistry is a name to type registry used to create objects by name.
type ClassRegistry struct {
	namedType map[string]reflect.Type
}

// Plugin is the global ClassRegistry singleton.
// Use this singleton to register plugins.
var Plugin = ClassRegistry{make(map[string]reflect.Type)}

// Register a plugin to the ClassRegistry by passing an uninitialized object.
// Example: var MyConsumerClassID = shared.Plugin.Register(MyConsumer{})
func (registry ClassRegistry) Register(typeInstance interface{}) {
	structType := reflect.TypeOf(typeInstance)
	pathIdx := strings.Index(structType.PkgPath(), gollumPackageName) + len(gollumPackageName)

	typeName := structType.PkgPath()[pathIdx:] + "." + structType.Name()
	registry.namedType[typeName] = structType
}

// Create an uninitialized object by class name.
// The class name has to be "package.class" or "package/subpackage.class".
// The gollum package is omitted from the package path.
func (registry ClassRegistry) Create(typeName string) (interface{}, reflect.Type, error) {
	structType, exists := registry.namedType[typeName]
	if exists {
		return reflect.New(structType).Elem().Interface(), structType, nil
	}
	return nil, nil, ClassRegistryError{"Unknown class: " + typeName}
}
