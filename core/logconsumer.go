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

package core

import (
	"sync"
)

// LogConsumer is an internal consumer plugin used indirectly by the gollum log
// package.
type LogConsumer struct {
	Consumer
	control   chan PluginControl
	logStream Stream
	sequence  uint64
}

// Configure initializes this consumer with values from a plugin config.
func (cons *LogConsumer) Configure(conf PluginConfigReader) error {
	cons.control = make(chan PluginControl, 1)
	cons.logStream = StreamRegistry.GetStream(LogInternalStreamID)
	return nil
}

// GetState always returns PluginStateActive
func (cons *LogConsumer) GetState() PluginState {
	return PluginStateActive
}

// Streams always returns an array with one member - the internal log stream
func (cons *LogConsumer) Streams() []MessageStreamID {
	return []MessageStreamID{LogInternalStreamID}
}

// Write fulfills the io.Writer interface
func (cons LogConsumer) Write(data []byte) (int, error) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	msg := NewMessage(cons, dataCopy, cons.sequence)
	msg.StreamID = LogInternalStreamID
	cons.logStream.Enqueue(msg)

	return len(data), nil
}

// Control returns a handle to the control channel
func (cons *LogConsumer) Control() chan<- PluginControl {
	return cons.control
}

// Consume starts listening for control statements
func (cons *LogConsumer) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == PluginControlStopConsumer {
			return // ### return ###
		}
	}
}
