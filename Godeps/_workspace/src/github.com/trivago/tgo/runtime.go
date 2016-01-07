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

// RecoverShutdown will trigger a shutdown via interrupt if a panic was issued.
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Print("Panic triggered shutdown: ", r)
		log.Print(string(debug.Stack()))

		// Send interrupt = clean shutdown
		// TODO: Hack
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}

// DontPanic can be used instead of RecoverShutdown when using a function
// without any parameters. E.g. go DontPanic(myFunction)
func DontPanic(callback func()) {
	defer RecoverShutdown()
	callback()
}
