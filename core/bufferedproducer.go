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

	"github.com/trivago/tgo"
)

// BufferedProducer plugin base type
//
// This type defines a common BufferedProducer base class.
// Producers may derive from this class.
//
// Parameters
//
// - Channel: This value defines the capacity of the message buffer.
// By default this parameter is set to "8192".
//
// - ChannelTimeoutMs: This value defines a timeout for each message
// before the message will discarded. To disable the timeout, set this
// parameter to 0.
// By default this parameter is set to "0".
//
//
type BufferedProducer struct {
	DirectProducer `gollumdoc:"embed_type"`
	messages       MessageQueue
	channelTimeout time.Duration `config:"ChannelTimeoutMs" default:"0" metric:"ms"`
}

// Configure initializes the standard producer config values.
func (prod *BufferedProducer) Configure(conf PluginConfigReader) {
	prod.onPrepareStop = prod.DefaultDrain
	prod.onStop = prod.DefaultClose
	prod.messages = NewMessageQueue(int(conf.GetInt("Channel", 8192)))
}

// GetQueueTimeout returns the duration this producer will block before a
// message is sent to the fallback. A value of -1 will cause the message to drop. A value
// of 0 will cause the producer to always block.
func (prod *BufferedProducer) GetQueueTimeout() time.Duration {
	return prod.channelTimeout
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop. A timeout value != nil will overwrite the channel
// timeout value for this call.
func (prod *BufferedProducer) Enqueue(msg *Message, timeout time.Duration) {
	defer prod.enqueuePanicHandling(msg)

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.TryFallback(msg)
		return // ### return, closing down ###
	}

	if !prod.HasContinueAfterModulate(msg) {
		return
	}

	// Allow timeout overwrite
	usedTimeout := prod.channelTimeout
	if timeout != 0 {
		usedTimeout = timeout
	}

	switch prod.messages.Push(msg, usedTimeout) {
	case MessageQueueTimeout:
		prod.TryFallback(msg)
		prod.setState(PluginStateWaiting)

	case MessageQueueDiscard:
		CountMessageDiscarded()
		prod.setState(PluginStateWaiting)

	default:
		prod.setState(PluginStateActive)
	}

	MessageTrace(msg, prod.GetID(), "Enqueued by buffered producer")
}

// DefaultDrain is the function registered to onPrepareStop by default.
// It calls DrainMessageChannel with the message handling function passed to
// Any of the control functions. If no such call happens, this function does
// nothing.
func (prod *BufferedProducer) DefaultDrain() {
	if prod.onMessage != nil {
		prod.DrainMessageChannel(prod.onMessage, prod.shutdownTimeout)
	}
}

// DrainMessageChannel empties the message channel. This functions returns
// after the queue being empty for a given amount of time or when the queue
// has been closed and no more messages are available. The return value
// indicates wether the channel is empty or not.
func (prod *BufferedProducer) DrainMessageChannel(handleMessage func(*Message), timeout time.Duration) bool {
	for {
		if msg, ok := prod.messages.PopWithTimeout(timeout); ok {
			if !tgo.ReturnAfter(prod.shutdownTimeout, func() { handleMessage(msg) }) {
				return false // ### return, done ###
			}
		} else {
			return prod.messages.IsEmpty() // ### return, done ###
		}
	}
}

// DefaultClose is the function registered to onStop by default.
// It calls CloseMessageChannel with the message handling function passed to
// Any of the control functions. If no such call happens, this function does
// nothing.
func (prod *BufferedProducer) DefaultClose() {
	if prod.onMessage != nil {
		prod.CloseMessageChannel(prod.onMessage)
	}
}

// CloseMessageChannel first calls DrainMessageChannel with shutdown timeout,
// closes the channel afterwards and calls DrainMessageChannel again to make
// sure all messages are actually gone. The return value indicates wether
// the channel is empty or not.
func (prod *BufferedProducer) CloseMessageChannel(handleMessage func(*Message)) (empty bool) {
	prod.DrainMessageChannel(handleMessage, prod.shutdownTimeout)
	prod.messages.Close()

	defer func() {
		if !prod.messages.IsEmpty() {
			prod.Logger.Errorf("%d messages left after closing.", prod.messages.GetNumQueued())
		}
	}()

	for {
		if msg, ok := prod.messages.Pop(); ok {
			if !tgo.ReturnAfter(prod.shutdownTimeout, func() { handleMessage(msg) }) {
				return false // ### return, failed to handle message ###
			}
		} else {
			return true // ### return, done ###
		}
	}
}

// MessageControlLoop provides a producer main loop that is sufficient for most
// use cases. ControlLoop will be called in a separate go routine.
// This function will block until a stop signal is received.
func (prod *BufferedProducer) MessageControlLoop(onMessage func(*Message)) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	prod.messageLoop(onMessage)
}

// TickerMessageControlLoop is like MessageLoop but executes a given function at
// every given interval tick, too. If the onTick function takes longer than
// interval, the next tick will be delayed until onTick finishes.
func (prod *BufferedProducer) TickerMessageControlLoop(onMessage func(*Message), interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	go prod.tickerLoop(interval, onTimeOut)
	prod.messageLoop(onMessage)
}

func (prod *BufferedProducer) messageLoop(onMessage func(*Message)) {
	prod.onMessage = onMessage
	for prod.IsActive() {
		msg, more := prod.messages.Pop()
		if more {
			onMessage(msg)
		}
	}
}
