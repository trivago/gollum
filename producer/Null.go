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

package producer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"sync"
)

// Null producer plugin
// Configuration example
//
//   - "producer.Null":
//     Enable: true
//
// This producer does nothing and provides only bare-bone configuration (i.e.
// enabled, channel and streams).
// Use this producer to test consumer performance.
type Null struct {
	control chan core.ProducerControl
	streams []core.MessageStreamID
}

func init() {
	shared.RuntimeType.Register(Null{})
}

// Configure initializes the basic members
func (prod *Null) Configure(conf core.PluginConfig) error {
	prod.control = make(chan core.ProducerControl, 1)
	prod.streams = make([]core.MessageStreamID, len(conf.Stream))
	return nil
}

// Streams returns the streams this producer is listening to.
func (prod Null) Streams() []core.MessageStreamID {
	return prod.streams
}

// Control returns write access to this producer's control channel.
func (prod Null) Control() chan<- core.ProducerControl {
	return prod.control
}

// Post simply ignores the message
func (prod Null) Post(msg core.Message) {
}

// Produce writes to a buffer that is dumped to a file.
func (prod Null) Produce(threads *sync.WaitGroup) {
	for {
		command := <-prod.control
		if command == core.ProducerControlStop {
			return // ### return ###
		}
	}
}
