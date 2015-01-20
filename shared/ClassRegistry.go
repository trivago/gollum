package shared

import (
	"hash/fnv"
	"reflect"
)

// Error returned by ClassRegistry functions
type ClassRegistryError struct {
	Message string
}

// Error interface implementation for ClassRegistryError
func (err ClassRegistryError) Error() string {
	return err.Message
}

// A name to type registry used to create objects by name
type ClassRegistry struct {
	namedType map[string]reflect.Type
}

// The global ClassRegistry singleton.
// Use this singleton to register plugins.
var Plugin ClassRegistry = ClassRegistry{make(map[string]reflect.Type)}

// Register a plugin to the ClassRegistry by passing an uninitialized object.
// Example: var MyConsumerClassID = shared.Plugin.Register(MyConsumer{})
func (registry ClassRegistry) Register(typeInstance interface{}) int {
	structType := reflect.TypeOf(typeInstance)
	typeName := structType.PkgPath()[len("gollum/"):] + "." + structType.Name()
	registry.namedType[typeName] = structType

	nameHash := fnv.New32a()
	nameHash.Write([]byte(typeName))
	return int(nameHash.Sum32())
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
