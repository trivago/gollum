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
	"fmt"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"sync"
	"time"
)

// Producer is an interface for plugins that pass messages to other services,
// files or storages.
type Producer interface {
	// Enqueue sends a message to the producer. The producer may reject
	// the message or drop it after a given timeout. Enqueue can block.
	Enqueue(msg Message, timeout *time.Duration)

	// Produce should implement a main loop that passes messages from the
	// message channel to some other service like the console.
	// This can be part of this function or a separate go routine.
	// Produce is always called as a go routine.
	Produce(workers *sync.WaitGroup)

	// Streams returns the streams this producer is listening to.
	Streams() []MessageStreamID

	// Control returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- PluginControl

	// DropStream returns the id of the stream to drop messages to.
	GetDropStreamID() MessageStreamID

	// Close closes and empties the internal channel and returns once all
	// messages have been processed.
	Close()
}

// ProducerBase base class
// All producers support a common subset of configuration options:
//
//   - "producer.Something":
//     Enable: true
//     Channel: 1024
//     ChannelTimeoutMs: 200
//     Formatter: "format.Envelope"
//     DropToStream: "failed"
//     Stream:
//       - "error"
//       - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 8192.
//
// ChannelTimeoutMs sets a timeout in milliseconds for messages to wait if this
// producer's queue is full.
// A timeout of -1 or lower will drop the message without notice.
// A timeout of 0 will block until the queue is free. This is the default.
// A timeout of 1 or higher will wait x milliseconds for the queues to become
// available again. If this does not happen, the message will be send to the
// retry channel.
//
// ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect
// a blocking producer during shutdown. By default this is set to 3 seconds.
// If processing a message takes longer to process than this duration, messages
// will be dropped during shutdown.
//
// Stream contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "listen to all streams but the internal".
//
// DroppedStream defines the stream used for messages that are dropped after
// a timeout (see ChannelTimeoutMs). By default this is _DROPPED_.
//
// Formatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward.
type ProducerBase struct {
	messages        chan Message
	control         chan PluginControl
	streams         []MessageStreamID
	dropStreamID    MessageStreamID
	state           *PluginRunState
	timeout         time.Duration
	shutdownTimeout time.Duration
	format          Formatter
	onRoll          func()
	onStop          func()
	onPrepareStop   func()
}

// ProducerError can be used to return consumer related errors e.g. during a
// call to Configure
type ProducerError struct {
	message string
}

// NewProducerError creates a new ProducerError
func NewProducerError(args ...interface{}) ProducerError {
	return ProducerError{fmt.Sprint(args...)}
}

// Error satisfies the error interface for the ProducerError struct
func (err ProducerError) Error() string {
	return err.message
}

// Configure initializes the standard producer config values.
func (prod *ProducerBase) Configure(conf PluginConfig) error {

	format, err := NewPluginWithType(conf.GetString("Formatter", "format.Forward"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}
	prod.format = format.(Formatter)

	prod.streams = make([]MessageStreamID, len(conf.Stream))
	prod.control = make(chan PluginControl, 1)
	prod.messages = make(chan Message, conf.GetInt("Channel", 8192))
	prod.timeout = time.Duration(conf.GetInt("ChannelTimeoutMs", 0)) * time.Millisecond
	prod.shutdownTimeout = time.Duration(conf.GetInt("ShutdownTimeoutMs", 3000)) * time.Millisecond
	prod.state = new(PluginRunState)
	prod.dropStreamID = GetStreamID(conf.GetString("DropToStream", DroppedStream))

	prod.onRoll = nil
	prod.onStop = nil
	prod.onPrepareStop = nil

	for i, stream := range conf.Stream {
		prod.streams[i] = GetStreamID(stream)
	}

	return nil
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (prod ProducerBase) SetRollCallback(onRoll func()) {
	prod.onRoll = onRoll
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (prod ProducerBase) SetStopCallback(onStop func()) {
	prod.onStop = onStop
}

// SetPrepareStopCallback sets the function to be called upon PluginControlPrepareStop
func (prod ProducerBase) SetPrepareStopCallback(onPrepareStop func()) {
	prod.onPrepareStop = onPrepareStop
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (prod ProducerBase) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	prod.state.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (prod ProducerBase) AddMainWorker(workers *sync.WaitGroup) {
	prod.state.SetWorkerWaitGroup(workers)
	prod.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (prod ProducerBase) AddWorker() {
	prod.state.AddWorker()
	shared.Metric.Inc(metricActiveWorkers)
}

// WorkerDone removes an additional worker to the waitgroup.
func (prod ProducerBase) WorkerDone() {
	prod.state.WorkerDone()
	shared.Metric.Dec(metricActiveWorkers)
}

// GetTimeout returns the duration this producer will block before a message
// is dropped. A value of -1 will cause the message to drop. A value of 0
// will cause the producer to always block.
func (prod ProducerBase) GetTimeout() time.Duration {
	return prod.timeout
}

// GetShutdownTimeout returns the duration this producer will wait during close
// before messages get dropped.
func (prod ProducerBase) GetShutdownTimeout() time.Duration {
	return prod.shutdownTimeout
}

// Next returns the latest message from the channel as well as the open state
// of the channel. This function blocks if the channel is empty.
func (prod ProducerBase) Next() (Message, bool) {
	msg, ok := <-prod.messages
	return msg, ok
}

// NextNonBlocking calls a given callback if a message is queued or returns.
// Returns false if no message was recieved.
func (prod ProducerBase) NextNonBlocking(onMessage func(msg Message)) bool {
	select {
	case msg := <-prod.messages:
		onMessage(msg)
		return true
	default:
		return false
	}
}

// Format calls the formatters Format function
func (prod *ProducerBase) Format(msg Message) ([]byte, MessageStreamID) {
	return prod.format.Format(msg)
}

// GetFormatter returns the formatter of this producer
func (prod *ProducerBase) GetFormatter() Formatter {
	return prod.format
}

// PauseAllStreams sends the Pause() command to all streams this producer is
// listening to.
func (prod *ProducerBase) PauseAllStreams(capacity int) {
	for _, streamID := range prod.streams {
		stream := StreamTypes.GetStream(streamID)
		stream.Pause(capacity)
	}
}

// ResumeAllStreams sends the Resume() command to all streams this producer is
// listening to.
func (prod *ProducerBase) ResumeAllStreams() {
	for _, streamID := range prod.streams {
		stream := StreamTypes.GetStream(streamID)
		stream.Resume()
	}
}

// Streams returns the streams this producer is listening to.
func (prod *ProducerBase) Streams() []MessageStreamID {
	return prod.streams
}

// GetDropStreamID returns the id of the stream to drop messages to.
func (prod *ProducerBase) GetDropStreamID() MessageStreamID {
	return prod.dropStreamID
}

// Control returns write access to this producer's control channel.
// See PluginControl* constants.
func (prod *ProducerBase) Control() chan<- PluginControl {
	return prod.control
}

// Messages returns write access to the message channel this producer reads from.
func (prod *ProducerBase) Messages() chan<- Message {
	return prod.messages
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop. A timeout value != nil will overwrite the channel
// timeout value for this call.
func (prod *ProducerBase) Enqueue(msg Message, timeout *time.Duration) {
	usedTimeout := prod.timeout
	if timeout != nil {
		usedTimeout = *timeout
	}
	switch msg.Enqueue(prod.messages, usedTimeout) {
	case MessageStateTimeout:
		prod.Drop(msg)

	case MessageStateDiscard:
		shared.Metric.Inc(MetricDiscarded)
	}
}

// Drop routes the message to the configured drop stream.
func (prod *ProducerBase) Drop(msg Message) {
	shared.Metric.Inc(MetricDropped)
	msg.Route(prod.dropStreamID)
}

// ProcessCommand provides a callback based possibility to react on the
// different producer commands. Returns true if ProducerControlStop was triggered.
func (prod *ProducerBase) processCommand(command PluginControl) bool {
	switch command {
	default:
		// Do nothing

	case PluginControlStop:
		if prod.onStop != nil {
			prod.onStop()
		}
		return true // ### return ###

	case PluginControlRoll:
		if prod.onRoll != nil {
			prod.onRoll()
		}

	case PluginControlPrepareStop:
		if prod.onPrepareStop != nil {
			prod.onPrepareStop()
		}
	}

	return false
}

// CloseGracefully closes and empties the internal channel with a given timeout
// per message. If flushing messages runs into this timeout all remaining
// messages will be dropped by using the Drop function.
// This method may be called from derived implementation's Close method.
// If a timout has been detected, false is returned.
func (prod *ProducerBase) CloseGracefully(onMessage func(msg Message)) bool {
	close(prod.messages)
	flushWorker := new(shared.WaitGroup)

	for msg := range prod.messages {
		// onMessage may block. To be able to exit this method we need to call
		// it async and wait for it to finish.

		flushWorker.Inc()
		go func() {
			defer flushWorker.Done()
			onMessage(msg)
		}()

		if !flushWorker.WaitFor(prod.shutdownTimeout) {
			Log.Warning.Printf("A producer listening to %s has found to be blocking during Close(). Dropping remaining messages.", StreamTypes.GetStreamName(prod.Streams()[0]))
			for msg := range prod.messages {
				prod.Drop(msg)
			}
			return false // ### return, timed out ###
		}
	}

	return true
}

// DefaultControlLoop provides a producer mainloop that is sufficient for most
// usecases. Before this function exits Close will be called.
func (prod *ProducerBase) DefaultControlLoop(onMessage func(msg Message)) {
	for {
		select {
		case command := <-prod.control:
			if prod.processCommand(command) {
				return // ### return ###
			}

		case msg := <-prod.messages:
			onMessage(msg)
		}
	}
}

// TickerControlLoop is like DefaultControlLoop but executes a given function at
// every given interval tick, too. Before this function exits Close will be called.
func (prod *ProducerBase) TickerControlLoop(interval time.Duration, onMessage func(msg Message), onTimeOut func()) {
	flushTicker := time.NewTicker(interval)
	for {
		select {
		case command := <-prod.control:
			if prod.processCommand(command) {
				return // ### return ###
			}

		case msg := <-prod.messages:
			onMessage(msg)

		case <-flushTicker.C:
			onTimeOut()
		}
	}
}
