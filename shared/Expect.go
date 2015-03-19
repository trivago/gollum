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
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

// Expect is a helper construct for unittesting
type Expect struct {
	t *testing.T
}

// NewExpect returns a new Expect struct bound to a unittest object
func NewExpect(t *testing.T) Expect {
	return Expect{t}
}

func (e Expect) print(message string) {
	_, file, line, _ := runtime.Caller(2)
	e.t.Errorf("%s(%d): %s", file, line, message)
}

func (e Expect) printf(format string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	e.t.Errorf(fmt.Sprintf("%s(%d): %s", file, line, format), args...)
}

// True tests if the given value is true
func (e Expect) True(val bool) bool {
	if !val {
		e.print("Expected true")
		return false
	}
	return true
}

// False tests if the given value is false
func (e Expect) False(val bool) bool {
	if val {
		e.print("Expected false")
		return false
	}
	return true
}

// Nil tests if the given value is nil
func (e Expect) Nil(val interface{}) bool {
	rval := reflect.ValueOf(val)
	if val != nil && !rval.IsNil() {
		e.print("Expected nil")
		return false
	}
	return true
}

// NotNil tests if the given value is not nil
func (e Expect) NotNil(val interface{}) bool {
	rval := reflect.ValueOf(val)
	if val == nil || rval.IsNil() {
		e.print("Expected not nil")
		return false
	}
	return true
}

// IntEq tests two integers on equality
func (e Expect) IntEq(val1 int, val2 int) bool {
	if val1 != val2 {
		e.printf("Expected \"%d\", got \"%d\"", val1, val2)
		return false
	}
	return true
}

// StringEq tests two strings on equality
func (e Expect) StringEq(val1 string, val2 string) bool {
	if val1 != val2 {
		e.printf("Expected \"%s\", got \"%s\"", val1, val2)
		return false
	}
	return true
}

// BytesEq tests two byte slices on equality
func (e Expect) BytesEq(val1 []byte, val2 []byte) bool {
	if !bytes.Equal(val1, val2) {
		e.printf("Expected %v, got %v", val1, val2)
		return false
	}
	return true
}

// MapSetStrEq tests if a key/value pair is set in a given map and if value is
// of the expected string value
func (e Expect) MapSetStrEq(data map[string]interface{}, key string, value string) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}

	stringVal, correctType := val.(string)
	if !correctType {
		e.printf("Key \"%s\" is not a string (%s)", key, reflect.TypeOf(val))
		return false
	}

	if stringVal != value {
		e.printf("Expected \"%s\" for \"%s\" got \"%s\"", value, key, stringVal)
		return false
	}
	return true
}

// MapSetIntEq tests if a key/value pair is set in a given map and if value is
// of the expected integer value
func (e Expect) MapSetIntEq(data map[string]interface{}, key string, value int) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}

	intVal, correctType := val.(int)
	if !correctType {
		e.printf("Key \"%s\" is not an int (%s)", key, reflect.TypeOf(val))
		return false
	}

	if intVal != value {
		e.printf("Expected \"%d\" for \"%s\" got \"%d\"", value, key, intVal)
		return false
	}
	return true
}

// MapSetFloat64Eq tests if a key/value pair is set in a given map and if value is
// of the expected integer value
func (e Expect) MapSetFloat64Eq(data map[string]interface{}, key string, value float64) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}

	floatVal, correctType := val.(float64)
	if !correctType {
		e.printf("Key \"%s\" is not a float64 (%s)", key, reflect.TypeOf(val))
		return false
	}

	if floatVal != value {
		e.printf("Expected \"%f\" for \"%s\" got \"%f\"", value, key, floatVal)
		return false
	}
	return true
}

// MapSetStrArrayEq tests if a key/value pair is set in a given map and if value
// is matching the expected array value
func (e Expect) MapSetStrArrayEq(data map[string]interface{}, key string, value []string) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}

	stringArrayVal, correctType := val.([]string)
	if !correctType {
		e.printf("Key \"%s\" is not an []string", key)
		return false
	}

	if len(value) != len(stringArrayVal) {
		e.printf("Expected \"%v\" for \"%s\" got \"%v\"", value, key, val)
		return false
	}

	for i, element := range stringArrayVal {
		if element != value[i] {
			e.printf("Expected \"%s\" at for \"%s[%d]\" got \"%s\"", element, key, i, value[i])
			return false
		}
	}
	return true
}

// MapSet tests if a given key exists in a map
func (e Expect) MapSet(data map[string]interface{}, key string) bool {
	_, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" to be set", key)
		return false
	}
	return true
}

// MapNotSet tests if a given key does not exist in a map
func (e Expect) MapNotSet(data map[string]interface{}, key string) bool {
	_, valSet := data[key]
	if valSet {
		e.printf("Expected key \"%s\" not to be set", key)
		return false
	}
	return true
}
