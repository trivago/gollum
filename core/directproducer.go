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
	"time"
)

// DirectProducer plugin base type
//
// This type defines a common baseclass for producers.
//
type DirectProducer struct {
	SimpleProducer `gollumdoc:"embed_type"`
	onMessage      func(*Message)
}

// Configure initializes the standard producer config values.
func (prod *DirectProducer) Configure(conf PluginConfigReader) {
	// We need to have an empty Configure method, otherwise
	// SimpleProducer.Configure will be called twice as Configure is inherited.
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

	if !prod.HasContinueAfterModulate(msg) {
		return
	}

	prod.onMessage(msg)
	MessageTrace(msg, prod.GetID(), "Enqueued by direct producer")
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
func (prod *DirectProducer) TickerMessageControlLoop(onMessage func(*Message),
	interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	prod.onMessage = onMessage
	prod.TickerControlLoop(interval, onTimeOut)
}

// default panic handling for Enqueue() function
func (prod *DirectProducer) enqueuePanicHandling(msg *Message) {
	if r := recover(); r != nil {
		prod.Logger.Error("Recovered a panic during producer enqueue: ", r)
		prod.Logger.Error("Producer: ", prod.id, "State: ", prod.GetState(),
			", Router: ", StreamRegistry.GetStreamName(msg.GetStreamID()))
		prod.TryFallback(msg)
	}
}
