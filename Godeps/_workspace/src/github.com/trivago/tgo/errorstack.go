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

package tgo

import (
	"fmt"
)

// ErrorStack is a helper to store errors from multiple statements for
// batch handling. Convenience functions to wrap function calls of the
// form func() (<type>, error) do exist for all golang base types.
type ErrorStack struct {
	errors []error
}

// NewErrorStack creates a new error stack
func NewErrorStack() ErrorStack {
	return ErrorStack{
		errors: []error{},
	}
}

// Push adds a new error to the top of the error stack.
// Returns if err != nil.
func (stack *ErrorStack) Push(err error) bool {
	if err != nil {
		stack.errors = append(stack.errors, err)
		return true
	}
	return false
}

// Pushf adds a new error message to the top of the error stack
func (stack *ErrorStack) Pushf(message string, args ...interface{}) {
	stack.errors = append(stack.errors, fmt.Errorf(message, args...))
}

// Pop removes an error from the top of the stack and returns it
func (stack *ErrorStack) Pop() error {
	if len(stack.errors) == 0 {
		return nil
	}
	err := stack.errors[len(stack.errors)-1]
	stack.errors = stack.errors[:len(stack.errors)]
	return err
}

// Top returns the error on top of the stack (last error pushed)
func (stack ErrorStack) Top() error {
	if len(stack.errors) == 0 {
		return nil
	}
	return stack.errors[len(stack.errors)-1]
}

// Error implements the Error interface
func (stack ErrorStack) Error() string {
	if len(stack.errors) == 0 {
		return ""
	}
	errString := ""
	for idx, err := range stack.errors {
		errString = fmt.Sprintf("%s%d: %s\n", errString, idx, err.Error())
	}
	return errString
}

// OrNil returns this object or nil of no errors are stored
func (stack *ErrorStack) OrNil() error {
	if len(stack.errors) == 0 {
		return nil
	}
	return stack
}

// Clear removes all errors from the stack
func (stack *ErrorStack) Clear() {
	stack.errors = []error{}
}

// Value can be used to wrap a function call returning an interface{} and an error,
// will Push the error and return the value
func (stack *ErrorStack) Value(value interface{}, err error) interface{} {
	stack.Push(err)
	return value
}

// Array can be used to wrap a function call returning an array and an error,
// will Push the error and return the value
func (stack *ErrorStack) Array(value []interface{}, err error) []interface{} {
	stack.Push(err)
	return value
}

// Map can be used to wrap a function call returning a map and an error,
// will Push the error and return the value
func (stack *ErrorStack) Map(value map[interface{}]interface{}, err error) map[interface{}]interface{} {
	stack.Push(err)
	return value
}

// Bool can be used to wrap a function call returning a bool and an error,
// will Push the error and return the value
func (stack *ErrorStack) Bool(value bool, err error) bool {
	stack.Push(err)
	return value
}

// Byte can be used to wrap a function call returning a byte and an error,
// will Push the error and return the value
func (stack *ErrorStack) Byte(value byte, err error) byte {
	stack.Push(err)
	return value
}

// Rune can be used to wrap a function call returning a rune and an error,
// will Push the error and return the value
func (stack *ErrorStack) Rune(value rune, err error) rune {
	stack.Push(err)
	return value
}

// Int can be used to wrap a function call returning an int and an error,
// will Push the error and return the value
func (stack *ErrorStack) Int(value int, err error) int {
	stack.Push(err)
	return value
}

// Uint can be used to wrap a function call returning an uint and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uint(value uint, err error) uint {
	stack.Push(err)
	return value
}

// Uintptr can be used to wrap a function call returning an uintptr and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uintptr(value uintptr, err error) uintptr {
	stack.Push(err)
	return value
}

// Int8 can be used to wrap a function call returning an int8 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Int8(value int8, err error) int8 {
	stack.Push(err)
	return value
}

// Uint8 can be used to wrap a function call returning an uint8 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uint8(value uint8, err error) uint8 {
	stack.Push(err)
	return value
}

// Int16 can be used to wrap a function call returning an int16 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Int16(value int16, err error) int16 {
	stack.Push(err)
	return value
}

// Uint16 can be used to wrap a function call returning an uint16 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uint16(value uint16, err error) uint16 {
	stack.Push(err)
	return value
}

// Int32 can be used to wrap a function call returning an int32 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Int32(value int32, err error) int32 {
	stack.Push(err)
	return value
}

// Uint32 can be used to wrap a function call returning an uint32 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uint32(value uint32, err error) uint32 {
	stack.Push(err)
	return value
}

// Int64 can be used to wrap a function call returning an int64 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Int64(value int64, err error) int64 {
	stack.Push(err)
	return value
}

// Uint64 can be used to wrap a function call returning an uint64 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Uint64(value uint64, err error) uint64 {
	stack.Push(err)
	return value
}

// Float32 can be used to wrap a function call returning an float32 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Float32(value float32, err error) float32 {
	stack.Push(err)
	return value
}

// Float64 can be used to wrap a function call returning an float64 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Float64(value float64, err error) float64 {
	stack.Push(err)
	return value
}

// Complex64 can be used to wrap a function call returning an complex64 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Complex64(value complex64, err error) complex64 {
	stack.Push(err)
	return value
}

// Complex128 can be used to wrap a function call returning an complex128 and an error,
// will Push the error and return the value
func (stack *ErrorStack) Complex128(value complex128, err error) complex128 {
	stack.Push(err)
	return value
}

// Str can be used to wrap a function call returning an string and an error,
// will Push the error and return the value
func (stack *ErrorStack) String(value string, err error) string {
	stack.Push(err)
	return value
}
