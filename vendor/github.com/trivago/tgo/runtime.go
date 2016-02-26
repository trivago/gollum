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
	"log"
	"os"
	"runtime/debug"
)

// ShutdownCallback holds the function that is called when RecoverShutdown detects
// a panic and could recover. By default this functions sends an os.Interrupt
// signal to the process. This function can be overwritten for customization.
var ShutdownCallback = func() {
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(os.Interrupt)
}

// RecoverShutdown will trigger a shutdown via os.Interrupt if a panic was issued.
// A callstack will be printed like with RecoverTrace().
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Print("Panic triggered shutdown: ", r)
		log.Print(string(debug.Stack()))
		ShutdownCallback()
	}
}

// RecoverTrace will trigger a stack trace when a panic was recovered by this
// function. Typically used as "defer RecoverTrace()".
func RecoverTrace() {
	if r := recover(); r != nil {
		log.Print("Panic ignored: ", r)
		log.Print(string(debug.Stack()))
	}
}

// WithRecover can be used instead of RecoverTrace when using a function
// without any parameters. E.g. "go WithRecover(myFunction)"
func WithRecover(callback func()) {
	defer RecoverTrace()
	callback()
}

// WithRecoverShutdown can be used instead of RecoverShutdown when using a function
// without any parameters. E.g. "go WithRecoverShutdown(myFunction)"
func WithRecoverShutdown(callback func()) {
	defer RecoverShutdown()
	callback()
}
