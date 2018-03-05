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
	"fmt"
)

// ModulateResultError is used by modulators to return a problem that happened
// during the modulation process, caused by the modulator.
type ModulateResultError struct {
	message string
}

// Error fullfills the golang error interface
func (p ModulateResultError) Error() string {
	return p.message
}

// NewModulateResultError creates a new ModulateResultError with the given
// message.
func NewModulateResultError(message string, values ...interface{}) ModulateResultError {
	return ModulateResultError{
		message: fmt.Sprintf(message, values),
	}
}
