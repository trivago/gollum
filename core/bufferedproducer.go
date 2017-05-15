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
	"github.com/trivago/tgo"
	"time"
)

// BufferedProducer plugin base type
// This type defines a common baseclass for producers. All producers should
// derive from this class, but not necessarily need to.
// Configuration example:
//
//  - "producer.Foobar":
//    Enable: true
//    ID: ""
//    Channel: 8192
//    ChannelTimeoutMs: 0
//    ShutdownTimeoutMs: 1000
//    Formatter: "format.Forward"
//    Filter: "filter.All"
//    FallbackStream: "_DROPPED_"
//    Router:
//      - "foo"
//      - "bar"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// ID allows this producer to be found by other plugins by name. By default this
// is set to "" which does not register this producer.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 8192.
//
// ChannelTimeoutMs sets a timeout in milliseconds for messages to wait if this
// producer's queue is full.
// A timeout of -1 or lower will send the message the the fallback without notice.
// A timeout of 0 will block until the queue is free. This is the default.
// A timeout of 1 or higher will wait x milliseconds for the queues to become
// available again. If this does not happen, the message will be send to the
// retry channel.
//
// ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect
// a blocking producer during shutdown. By default this is set to 1 second.
// Decreasing this value may lead to lost messages during shutdown. Increasing
// this value will increase shutdown time.
//
// Router contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "listen to all routers but the internal".
//
// FallbackStream defines the stream used for messages that are sent to the fallback after
// a timeout (see ChannelTimeoutMs). By default this is _DROPPED_.
//
// Formatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward.
// Each producer decides if and when to use a Formatter.
//
// Filter sets a filter that is applied before formatting, i.e. before a message
// is send to the message queue. If a producer requires filtering after
// formatting it has to define a separate filter as the producer decides if
// and where to format.
//
type BufferedProducer struct {
	SimpleProducer
	messages       MessageQueue
	channelTimeout time.Duration
	onMessage      func(*Message)
}

// Configure initializes the standard producer config values.
func (prod *BufferedProducer) Configure(conf PluginConfigReader) error {
	prod.SimpleProducer.Configure(conf)
	prod.onPrepareStop = prod.DefaultDrain
	prod.onStop = prod.DefaultClose

	prod.messages = NewMessageQueue(conf.GetInt("Channel", 8192))
	prod.channelTimeout = time.Duration(conf.GetInt("ChannelTimeoutMs", 0)) * time.Millisecond

	return conf.Errors.OrNil()
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
func (prod *BufferedProducer) Enqueue(msg *Message, timeout *time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			prod.Log.Error.Print("Recovered a panic during producer enqueue: ", r)
			prod.Log.Error.Print("Producer: ", prod.id, "State: ", prod.GetState(), ", Router: ", StreamRegistry.GetStreamName(msg.StreamID()))
			prod.TryFallback(msg)
		}
	}()

	// Run modulators and drop message if necessary
	result := prod.Modulate(msg)
	switch result {
	case ModulateResultDiscard:
		DiscardMessage(msg)
		return

	case ModulateResultFallback:
		RouteOriginal(msg)
		return

	case ModulateResultContinue:
		// OK

	default:
		prod.Log.Error.Print("Modulator result not supported:", result)
		return
	}

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.TryFallback(msg)
		return // ### return, closing down ###
	}

	// Allow timeout overwrite
	usedTimeout := prod.channelTimeout
	if timeout != nil {
		usedTimeout = *timeout
	}

	switch prod.messages.Push(msg, usedTimeout) {
	case MessageStateTimeout:
		prod.TryFallback(msg)
		prod.setState(PluginStateWaiting)

	case MessageStateDiscard:
		CountDiscardedMessage()
		prod.setState(PluginStateWaiting)

	default:
		prod.setState(PluginStateActive)
	}
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
			prod.Log.Error.Printf("%d messages left after closing.", prod.messages.GetNumQueued())
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

// MessageControlLoop provides a producer mainloop that is sufficient for most
// usecases. ControlLoop will be called in a separate go routine.
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
