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

package shared

import (
	"fmt"
	"reflect"
	"strings"
)

// typeRegistry is a name to type registry used to create objects by name.
type typeRegistry struct {
	namedType map[string]reflect.Type
}

// TypeRegistry is the global typeRegistry instance.
// Use this instance to register plugins.
var TypeRegistry = typeRegistry{
	namedType: make(map[string]reflect.Type),
}

// Register a plugin to the typeRegistry by passing an uninitialized object.
// Example: var MyConsumerClassID = shared.Plugin.Register(MyConsumer{})
func (registry typeRegistry) Register(typeInstance interface{}) {
	structType := reflect.TypeOf(typeInstance)
	packageName := structType.PkgPath()
	typeName := structType.Name()

	for n := 1; n < 5; n++ {
		pathIdx := LastIndexN(packageName, "/", n)
		if pathIdx == -1 {
			return // ### return, full path stored ###
		}
		shortPath := strings.Replace(packageName[pathIdx+1:], "/", ".", -1)
		shortTypeName := shortPath + "." + typeName
		registry.namedType[shortTypeName] = structType
	}
}

// New creates an uninitialized object by class name.
// The class name has to be "package.class" or "package/subpackage.class".
// The gollum package is omitted from the package path.
func (registry typeRegistry) New(typeName string) (interface{}, error) {
	structType, exists := registry.namedType[typeName]
	if exists {
		return reflect.New(structType).Interface(), nil
	}
	return nil, fmt.Errorf("Unknown class: %s", typeName)
}

// GetTypeOf returns only the type asscociated with the given name.
// If the name is not registered, nil is returned.
// The type returned will be a pointer type.
func (registry typeRegistry) GetTypeOf(typeName string) reflect.Type {
	if structType, exists := registry.namedType[typeName]; exists {
		return reflect.PtrTo(structType)
	}
	return nil
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
