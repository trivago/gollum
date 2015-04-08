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

package core

import (
	"sync"
)

// LogConsumer is an internal consumer plugin used indirectly by the gollum log
// package.
type LogConsumer struct {
	control   chan ConsumerControl
	logStream Stream
	sequence  uint64
}

// Configure initializes this consumer with values from a plugin config.
func (cons *LogConsumer) Configure(conf PluginConfig) error {
	cons.control = make(chan ConsumerControl, 1)
	cons.logStream = StreamTypes.GetStream(LogInternalStreamID)
	return nil
}

// Streams always returns an array with one member - the internal log stream
func (cons *LogConsumer) Streams() []MessageStreamID {
	return []MessageStreamID{LogInternalStreamID}
}

// Write fullfills the io.Writer interface
func (cons LogConsumer) Write(data []byte) (int, error) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	cons.logStream.Enqueue(NewMessage(cons, dataCopy, cons.sequence))
	return len(data), nil
}

// Pause is not implemented
func (cons LogConsumer) Pause() {
}

// IsPaused returns false as Pause is not implemented
func (cons LogConsumer) IsPaused() bool {
	return false
}

// Resume is not implemented as Pause is not implemented.
func (cons LogConsumer) Resume() {
}

// Control returns a handle to the control channel
func (cons *LogConsumer) Control() chan<- ConsumerControl {
	return cons.control
}

// Consume starts listening for control statements
func (cons *LogConsumer) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == ConsumerControlStop {
			return // ### return ###
		}
	}
}
