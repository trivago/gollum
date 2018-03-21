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
	"sync"
	"time"
)

// Producer is an interface for plugins that pass messages to other services,
// files or storages.
type Producer interface {
	PluginWithState
	MessageSource

	// Enqueue sends a message to the producer. The producer may reject
	// the message or try a fallback after a given timeout. Enqueue can block.
	Enqueue(msg *Message, timeout time.Duration)

	// Produce should implement a main loop that passes messages from the
	// message channel to some other service like the console.
	// This can be part of this function or a separate go routine.
	// Produce is always called as a go routine.
	Produce(workers *sync.WaitGroup)

	// Streams returns the streamIDs this producer is listening to.
	Streams() []MessageStreamID

	// Control returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- PluginControl

	// GetShutdownTimeout returns the duration gollum will wait for this producer
	// before canceling the shutdown process.
	GetShutdownTimeout() time.Duration
}
