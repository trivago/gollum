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

// Consumer is an interface for plugins that recieve data from outside sources
// and generate Message objects from this data.
type Consumer interface {
	// Consume should implement to main loop that fetches messages from a given
	// source and pushes it to the Message channel.
	Consume(*sync.WaitGroup)

	// Streams returns the streams this consumer is writing to.
	Streams() []MessageStreamID

	// Control returns write access to this consumer's control channel.
	// See PluginControl* constants.
	Control() chan<- PluginControl
}

// ConsumerBase base class
// All consumers support a common subset of configuration options:
//
// - "consumer.Something":
//   Enable: true
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Stream contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all streams" will get these
// messages.
type ConsumerBase struct {
	control chan PluginControl
	streams []MappedStream
	state   *PluginRunState
	timeout time.Duration
}

// ConsumerError can be used to return consumer related errors e.g. during a
// call to Configure
type ConsumerError struct {
	message string
}

// NewConsumerError creates a new ConsumerError
func NewConsumerError(args ...interface{}) ConsumerError {
	return ConsumerError{fmt.Sprint(args...)}
}

// Error satisfies the error interface for the ConsumerError struct
func (err ConsumerError) Error() string {
	return err.message
}

// Configure initializes standard consumer values from a plugin config.
func (cons *ConsumerBase) Configure(conf PluginConfig) error {
	cons.control = make(chan PluginControl, 1)
	cons.timeout = time.Duration(conf.GetInt("ChannelTimeout", 0)) * time.Millisecond
	cons.state = new(PluginRunState)

	for _, streamName := range conf.Stream {
		streamID := GetStreamID(streamName)
		cons.streams = append(cons.streams, MappedStream{
			StreamID: streamID,
			Stream:   StreamTypes.GetStreamOrFallback(streamID),
		})
	}

	return nil
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (cons ConsumerBase) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	cons.state.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (cons ConsumerBase) AddMainWorker(workers *sync.WaitGroup) {
	cons.state.SetWorkerWaitGroup(workers)
	cons.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (cons ConsumerBase) AddWorker() {
	cons.state.AddWorker()
	shared.Metric.Inc(metricActiveWorkers)
}

// WorkerDone removes an additional worker to the waitgroup.
func (cons ConsumerBase) WorkerDone() {
	cons.state.WorkerDone()
	shared.Metric.Dec(metricActiveWorkers)
}

// Enqueue creates a new message from a given byte slice and passes it to
// EnqueueMessage. Note that data is not copied, just referenced by the message.
func (cons *ConsumerBase) Enqueue(data []byte, sequence uint64) {
	msg := NewMessage(cons, data, sequence)
	cons.EnqueueMessage(msg)
}

// EnqueueCopy behaves like Enqueue but creates a copy of data that is attached
// to the message.
func (cons *ConsumerBase) EnqueueCopy(data []byte, sequence uint64) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	cons.Enqueue(dataCopy, sequence)
}

// EnqueueMessage passes a given message  to all streams.
// Only the StreamID of the message is modified, everything else is passed as-is.
func (cons *ConsumerBase) EnqueueMessage(msg Message) {
	for _, mapping := range cons.streams {
		msg.StreamID = mapping.StreamID
		mapping.Stream.Enqueue(msg)
	}
}

// Streams returns an array with all stream ids this consumer is writing to.
func (cons *ConsumerBase) Streams() []MessageStreamID {
	streamIDs := make([]MessageStreamID, 0, len(cons.streams))
	for _, mapping := range cons.streams {
		streamIDs = append(streamIDs, mapping.StreamID)
	}
	return streamIDs
}

// Control returns write access to this consumer's control channel.
// See ConsumerControl* constants.
func (cons *ConsumerBase) Control() chan<- PluginControl {
	return cons.control
}

// ProcessCommand provides a callback based possibility to react on the
// different consumer commands. Returns true if ConsumerControlStop was triggered.
func (cons *ConsumerBase) ProcessCommand(command PluginControl, onRoll func()) bool {
	switch command {
	default:
		// Do nothing
	case PluginControlStop:
		return true // ### return ###
	case PluginControlRoll:
		if onRoll != nil {
			onRoll()
		}
	}

	return false
}

// DefaultControlLoop provides a consumer mainloop that is sufficient for most
// usecases.
func (cons *ConsumerBase) DefaultControlLoop(onRoll func()) {
	for {
		command := <-cons.control
		if cons.ProcessCommand(command, onRoll) {
			return // ### return ###
		}
	}
}

// TickerControlLoop is like DefaultControlLoop but executes a given function at
// every given interval tick, too.
func (cons *ConsumerBase) TickerControlLoop(interval time.Duration, onRoll func(), onTick func()) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case command := <-cons.control:
			if cons.ProcessCommand(command, onRoll) {
				return // ### return ###
			}
		case <-ticker.C:
			onTick()
		}
	}
}
