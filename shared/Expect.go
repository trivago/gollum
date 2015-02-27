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

// MapSetEq tests if a key/value pair is set in a given map and if value is of
// an expected value
func (e Expect) MapSetEq(data map[string]interface{}, key string, value string) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}
	if val != value {
		e.printf("Expected \"%s\" for \"%s\" got \"%s\"", value, key, val)
		return false
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
