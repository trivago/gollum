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
	"github.com/trivago/gollum/shared"
	"sync"
	"time"
)

// ProducerControl is an enumeration used by the Producer.Control() channel
type ProducerControl int

const (
	// ProducerControlStop will cause the producer to halt and shutdown.
	ProducerControlStop = ProducerControl(1)

	// ProducerControlRoll notifies the consumer about a log rotation or
	// revalidation/reconnect of the write target
	ProducerControlRoll = ProducerControl(2)
)

// Producer is an interface for plugins that pass Message objects to other
// services, files or storages.
type Producer interface {
	// Enqueue tries to send a message to the producer. The producer may reject
	// the message, may block or may drop the message after a given timeout.
	// The actual behavior is specific to the producer implementation.
	Enqueue(msg Message)

	// Produce should implement the main loop that passes messages from the
	// message channel to some other service like the console.
	Produce(*sync.WaitGroup)

	// Streams returns the streams this producer is listening to.
	Streams() []MessageStreamID

	// Control returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- ProducerControl
}

// ProducerBase base class
// All producers support a common subset of configuration options:
//
// - "producer.Something":
//   Enable: true
//   Channel: 1024
//   ChannelTimeout: 200
//   Formatter: "format.Envelope"
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 1024.
//
// ChannelTimeout sets a timeout for messages to wait if this producer's queue
// is full.
// A timeout of -1 or lower will drop the message without notice.
// A timeout of 0 will block until the queue is free. This is the default.
// A timeout of 1 or higher will wait x milliseconds for the queues to become
// available again. If this does not happen, the message will be send to the
// retry channel.
//
// Stream contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "all streams".
//
// Fromatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Envelope
type ProducerBase struct {
	messages chan Message
	control  chan ProducerControl
	streams  []MessageStreamID
	state    *PluginRunState
	timeout  time.Duration
	Format   Formatter
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
	prod.Format = format.(Formatter)

	prod.streams = make([]MessageStreamID, len(conf.Stream))
	prod.control = make(chan ProducerControl, 1)
	prod.messages = make(chan Message, conf.GetInt("Channel", 8192))
	prod.timeout = time.Duration(conf.GetInt("ChannelTimeout", 0)) * time.Millisecond
	prod.state = new(PluginRunState)

	for i, stream := range conf.Stream {
		prod.streams[i] = GetStreamID(stream)
	}

	return nil
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

// Streams returns the streams this producer is listening to.
func (prod *ProducerBase) Streams() []MessageStreamID {
	return prod.streams
}

// Control returns write access to this producer's control channel.
// See ProducerControl* constants.
func (prod *ProducerBase) Control() chan<- ProducerControl {
	return prod.control
}

// Messages returns write access to the message channel this producer reads from.
func (prod *ProducerBase) Messages() chan<- Message {
	return prod.messages
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop.
func (prod *ProducerBase) Enqueue(msg Message) {
	msg.Enqueue(prod.messages, prod.timeout)
}

// ProcessCommand provides a callback based possibility to react on the
// different producer commands. Returns true if ProducerControlStop was triggered.
func (prod *ProducerBase) ProcessCommand(command ProducerControl, onRoll func()) bool {
	switch command {
	default:
		// Do nothing
	case ProducerControlStop:
		return true // ### return ###
	case ProducerControlRoll:
		if onRoll != nil {
			onRoll()
		}
	}

	return false
}

// DefaultControlLoop provides a producer mainloop that is sufficient for most
// usecases.
func (prod *ProducerBase) DefaultControlLoop(onMessage func(msg Message), onRoll func()) {
	for {
		select {
		case message := <-prod.messages:
			onMessage(message)

		case command := <-prod.control:
			if prod.ProcessCommand(command, onRoll) {
				return // ### return ###
			}
		}
	}
}

// TickerControlLoop is like DefaultControlLoop but executes a given function at
// every given interval tick, too.
func (prod *ProducerBase) TickerControlLoop(interval time.Duration, onMessage func(msg Message), onRoll func(), onTimeOut func()) {
	flushTicker := time.NewTicker(interval)

	for {
		select {
		case message := <-prod.messages:
			onMessage(message)

		case command := <-prod.control:
			if prod.ProcessCommand(command, onRoll) {
				return // ### return ###
			}

		case <-flushTicker.C:
			onTimeOut()
		}
	}
}
