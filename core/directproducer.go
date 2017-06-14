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
	"time"
)

// DirectProducer plugin base type
// This type defines a common baseclass for producers. All producers should
// derive from this class, but not necessarily need to.
// Configuration example:
//
//  - "producer.Foobar":
//    Enable: true
//    ID: ""
//    ShutdownTimeoutMs: 1000
//    Modulators:
//      - format.Forward
//      - filter.All
//    FallbackStream: "_DROPPED_"
//    Streams:
//      - "foo"
//      - "bar"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// ID allows this producer to be found by other plugins by name. By default this
// is set to "" which does not register this producer.
//
// ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect
// a blocking producer during shutdown. By default this is set to 1 second.
// Decreasing this value may lead to lost messages during shutdown. Increasing
// this value will increase shutdown time.
//
// Streams contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "listen to all routers but the internal".
//
// FallbackStream defines the stream used for messages that are sent to the fallback after
// a timeout (see ChannelTimeoutMs). By default this is _DROPPED_.
//
// Modulators sets formatter and filter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward.
// Each producer decides if and when to use a Formatter.
//
//
type DirectProducer struct {
	SimpleProducer
	onMessage func(*Message)
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop. A timeout value != nil will overwrite the channel
// timeout value for this call.
func (prod *DirectProducer) Enqueue(msg *Message, timeout time.Duration) {
	defer prod.enqueuePanicHandling(msg)

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.TryFallback(msg)
		return // ### return, closing down ###
	}

	if prod.HasContinueAfterModulate(msg) == false {
		return
	}

	prod.onMessage(msg)
}

// MessageControlLoop provides a producer main loop that is sufficient for most
// use cases. ControlLoop will be called in a separate go routine.
// This function will block until a stop signal is received.
func (prod *DirectProducer) MessageControlLoop(onMessage func(*Message)) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	prod.onMessage = onMessage
}

// TickerMessageControlLoop is like MessageLoop but executes a given function at
// every given interval tick, too. If the onTick function takes longer than
// interval, the next tick will be delayed until onTick finishes.
func (prod *DirectProducer) TickerMessageControlLoop(onMessage func(*Message), interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	prod.onMessage = onMessage
	prod.TickerControlLoop(interval, onTimeOut)
}

// default panic handling for Enqueue() function
func (prod *DirectProducer) enqueuePanicHandling(msg *Message) {
	if r := recover(); r != nil {
		prod.Log.Error.Print("Recovered a panic during producer enqueue: ", r)
		prod.Log.Error.Print("Producer: ", prod.id, "State: ", prod.GetState(), ", Router: ", StreamRegistry.GetStreamName(msg.GetStreamID()))
		prod.TryFallback(msg)
	}
}
