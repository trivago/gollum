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

package Log

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Consumer consumer plugin
// This is an internal plugin to route go log messages into gollum
type Consumer struct {
	control chan shared.ConsumerControl
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Consumer) Configure(conf shared.PluginConfig) error {
	cons.control = make(chan shared.ConsumerControl, 1)
	return nil
}

// Control returns a handle to the control channel
func (cons Consumer) Control() chan<- shared.ConsumerControl {
	return cons.control
}

// Messages reroutes Log.Messages()
func (cons Consumer) Messages() <-chan shared.Message {
	return Messages()
}

// Consume starts listening for control statements
func (cons Consumer) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
