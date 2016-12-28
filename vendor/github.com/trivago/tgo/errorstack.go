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

package tgo

import (
	"fmt"
)

// ErrorStack is a helper to store errors from multiple statements for
// batch handling. Convenience functions to wrap function calls of the
// form func() (<type>, error) do exist for all golang base types.
type ErrorStack struct {
	errors []error
	format ErrorStackFormat
}

type ErrorStackFormat int

const (
	// ErrorStackFormatNumbered formats like "0: error\n..."
	ErrorStackFormatNumbered = ErrorStackFormat(iota)
	// ErrorStackFormatNewline formats like "error\n..."
	ErrorStackFormatNewline = ErrorStackFormat(iota)
	// ErrorStackFormatCSV formats like "error, ..."
	ErrorStackFormatCSV = ErrorStackFormat(iota)
)

// NewErrorStack creates a new error stack
func NewErrorStack() ErrorStack {
	return ErrorStack{
		errors: []error{},
		format: ErrorStackFormatNumbered,
	}
}

// SetFormat set the format used when Error() is called.
func (stack *ErrorStack) SetFormat(format ErrorStackFormat) {
	stack.format = format
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

// PushAndDescribe behaves like Push but allows to prepend a text before
// the error messages returned by err. The type of err will be lost.
func (stack *ErrorStack) PushAndDescribe(message string, err error) bool {
	if err != nil {
		stack.errors = append(stack.errors, fmt.Errorf(message+" %s", err.Error()))
		return true
	}
	return false
}

// Pop removes an error from the top of the stack and returns it
func (stack *ErrorStack) Pop() error {
	if len(stack.errors) == 0 {
		return nil
	}
	err := stack.errors[len(stack.errors)-1]
	stack.errors = stack.errors[:len(stack.errors)-1]
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
		switch stack.format {
		case ErrorStackFormatNumbered:
			errString = fmt.Sprintf("%s%d: %s\n", errString, idx, err.Error())
		case ErrorStackFormatNewline:
			errString = fmt.Sprintf("%s%s\n", errString, err.Error())
		case ErrorStackFormatCSV:
			errString = fmt.Sprintf("%s%s, ", errString, err.Error())
		}
	}
	return errString
}

// Errors returns all gathered errors as an array
func (stack ErrorStack) Errors() []error {
	return stack.errors
}

// Len returns the number of error on the stack
func (stack ErrorStack) Len() int {
	return len(stack.errors)
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
