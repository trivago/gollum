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

package shared

import (
	"log"
	"regexp"
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
	// Produce should implement the main loop that passes messages from the
	// message channel to some other service like the console.
	Produce(*sync.WaitGroup)

	// IsActive returns true if the producer is ready to accept new data.
	IsActive() bool

	// GetTimeout returns the duration this producer will block before a message
	// is dropped. A value of -1 will cause the message to drop. A value of 0
	// will cause the producer to always block.
	GetTimeout() time.Duration

	// Accepts returns true if the message is allowed to be send to this producer.
	Accepts(message Message) bool

	// Control returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- ProducerControl

	// Messages returns write access to the message channel this producer reads from.
	Messages() chan<- Message
}

// ProducerBase base class
// All producers support a common subset of configuration options:
//
// - "producer.Something":
//   Enable: true
//   Channel: 1024
//   Formatter: "format.Timestamp"
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 1024.
//
// Stream contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "all streams".
//
// Fromatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward
type ProducerBase struct {
	messages chan Message
	control  chan ProducerControl
	filter   *regexp.Regexp
	state    *PluginRunState
	format   Formatter
	timeout  time.Duration
}

// ProducerError can be used to return consumer related errors e.g. during a
// call to Configure
type ProducerError struct {
	message string
}

// NewProducerError creates a new ProducerError
func NewProducerError(message string) ProducerError {
	return ProducerError{message}
}

// Error satisfies the error interface for the ProducerError struct
func (err ProducerError) Error() string {
	return err.message
}

// Configure initializes the standard producer config values.
func (prod *ProducerBase) Configure(conf PluginConfig) error {
	prod.messages = make(chan Message, conf.Channel)
	prod.control = make(chan ProducerControl, 1)
	prod.filter = nil
	prod.timeout = time.Duration(conf.GetInt("ChannelTimeout", 0)) * time.Millisecond
	prod.state = new(PluginRunState)

	plugin, err := RuntimeType.NewPlugin(conf.GetString("Formatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	prod.format = plugin.(Formatter)

	filter := conf.GetString("Filter", "")
	if filter != "" {
		prod.filter, err = regexp.Compile(filter)
		if err != nil {
			log.Print("Regex error: ", err)
		}
	}

	return nil
}

// SetWaitGroup sets the given waitgroup. This is also done by MarkAsActive so
// it is only needed when AddWorker is called before MarkAsActive.
func (prod *ProducerBase) SetWaitGroup(threads *sync.WaitGroup) {
	prod.state.WaitGroup = threads
}

// MarkAsActive adds this producer to the wait group and marks it as active
func (prod *ProducerBase) MarkAsActive(threads *sync.WaitGroup) {
	prod.state.WaitGroup = threads
	prod.state.WaitGroup.Add(1)
	prod.state.Active = true
}

// MarkAsDone removes the producer from the wait group and marks it as inactive
func (prod ProducerBase) MarkAsDone() {
	prod.state.WaitGroup.Done()
	prod.state.Active = false
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (prod ProducerBase) AddWorker() {
	prod.state.WaitGroup.Add(1)
}

// WorkerDone removes an additional worker to the waitgroup.
func (prod ProducerBase) WorkerDone() {
	prod.state.WaitGroup.Done()
}

// IsActive returns true if the producer is ready to generate messages.
func (prod ProducerBase) IsActive() bool {
	return prod.state.Active
}

// GetTimeout returns the duration this producer will block before a message
// is dropped. A value of -1 will cause the message to drop. A value of 0
// will cause the producer to always block.
func (prod ProducerBase) GetTimeout() time.Duration {
	return prod.timeout
}

// Accepts returns true if the message matches a configured regexp or if no
// regexp is set in the config
func (prod ProducerBase) Accepts(message Message) bool {
	if prod.filter == nil {
		return true // ### return, pass everything ###
	}

	return prod.filter.MatchString(string(message.Data))
}

// Formatter returns the formatter configured with this producer
func (prod ProducerBase) Formatter() Formatter {
	return prod.format
}

// Next returns the next message from the queue. Blocks if the queue is empty.
func (prod ProducerBase) Next() Message {
	return <-prod.messages
}

// NextClosed is like Next but also returns the closed state of the channel
func (prod ProducerBase) NextClosed() (Message, bool) {
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

// Control returns write access to this producer's control channel.
// See ProducerControl* constants.
func (prod ProducerBase) Control() chan<- ProducerControl {
	return prod.control
}

// Messages returns write access to the message channel this producer reads from.
func (prod ProducerBase) Messages() chan<- Message {
	return prod.messages
}

// ProcessCommand provides a callback based possibility to react on the
// different producer commands. Returns true if ProducerControlStop was triggered.
func (prod ProducerBase) ProcessCommand(command ProducerControl, onRoll func()) bool {
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
// usecases. It marks this producer as active and loops over ProcessCommand
// as long as the producer is marked as active.
func (prod ProducerBase) DefaultControlLoop(threads *sync.WaitGroup, onMessage func(msg Message), onRoll func()) {
	prod.MarkAsActive(threads)

	for prod.IsActive() {
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
func (prod ProducerBase) TickerControlLoop(threads *sync.WaitGroup, interval time.Duration, onMessage func(msg Message), onRoll func(), onTimeOut func()) {
	flushTicker := time.NewTicker(interval)
	prod.MarkAsActive(threads)

	for prod.IsActive() {
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
