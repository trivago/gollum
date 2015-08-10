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
	PluginWithState
	MessageSource

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

	// AddDependency is called whenever a producer is registered that sends
	// messages to this producer. Dependencies are used to resolve shutdown
	// conflicts.
	AddDependency(Producer)

	// DependsOn returns true if this plugin has a direct or indirect dependency
	// on the given producer
	DependsOn(Producer) bool
}

// ProducerBase base class
// All producers support a common subset of configuration options:
//
//   - "producer.Something":
//     Enable: true
//	   ID: ""
//     Channel: 8192
//     ChannelTimeoutMs: 0
//     ShutdownTimeoutMs: 3000
//     Formatter: "format.Forward"
//     DropToStream: "_DROPPED_"
//     Stream:
//       - "console"
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
// DropToStream defines the stream used for messages that are dropped after
// a timeout (see ChannelTimeoutMs). By default this is _DROPPED_.
//
// Formatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward.
type ProducerBase struct {
	messages        chan Message
	control         chan PluginControl
	streams         []MessageStreamID
	dropStreamID    MessageStreamID
	dependencies    []Producer
	runState        *PluginRunState
	timeout         time.Duration
	shutdownTimeout time.Duration
	format          Formatter
	onRoll          func()
	onStop          func()
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
	prod.runState = NewPluginRunState()
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
	prod.dropStreamID = GetStreamID(conf.GetString("DropToStream", DroppedStream))

	prod.onRoll = nil
	prod.onStop = nil

	for i, stream := range conf.Stream {
		prod.streams[i] = GetStreamID(stream)
	}

	return nil
}

// setState sets the runstate of this plugin
func (prod *ProducerBase) setState(state PluginState) {
	prod.runState.state = state
}

// GetState returns the state this plugin is currently in
func (prod *ProducerBase) GetState() PluginState {
	return prod.runState.state
}

// IsActive returns true if GetState() returns active
func (prod *ProducerBase) IsActive() bool {
	return prod.GetState() == PluginStateActive
}

// IsActiveOrStopping returns true if GetState() returns active or stopping
func (prod *ProducerBase) IsActiveOrStopping() bool {
	return prod.GetState() == PluginStateActive || prod.GetState() == PluginStateStopping
}

// IsBlocked returns true if GetState() returns waiting
func (prod *ProducerBase) IsBlocked() bool {
	return prod.GetState() == PluginStateWaiting
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (prod *ProducerBase) SetRollCallback(onRoll func()) {
	prod.onRoll = onRoll
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (prod *ProducerBase) SetStopCallback(onStop func()) {
	prod.onStop = onStop
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (prod *ProducerBase) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	prod.runState.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (prod *ProducerBase) AddMainWorker(workers *sync.WaitGroup) {
	prod.runState.SetWorkerWaitGroup(workers)
	prod.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (prod *ProducerBase) AddWorker() {
	prod.runState.AddWorker()
}

// WorkerDone removes an additional worker to the waitgroup.
func (prod *ProducerBase) WorkerDone() {
	prod.runState.WorkerDone()
}

// AddDependency is called whenever a producer is registered that sends
// messages to this producer. Dependencies are used to resolve shutdown
// conflicts.
func (prod *ProducerBase) AddDependency(dep Producer) {
	for _, storedDep := range prod.dependencies {
		if storedDep == dep {
			return // ### return, duplicate ###
		}
	}
	prod.dependencies = append(prod.dependencies, dep)
}

// DependsOn returns true if this plugin has a direct or indirect dependency
// on the given producer
func (prod *ProducerBase) DependsOn(dep Producer) bool {
	for _, storedDep := range prod.dependencies {
		if storedDep == dep {
			return true // ### return, depends ###
		}
		if storedDep.DependsOn(dep) {
			return true // ### return, nested dependency ###
		}
	}
	return false
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
func (prod *ProducerBase) Next() (Message, bool) {
	msg, ok := <-prod.messages
	return msg, ok
}

// NextNonBlocking calls a given callback if a message is queued or returns.
// Returns false if no message was recieved.
func (prod *ProducerBase) NextNonBlocking(onMessage func(msg Message)) bool {
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
		stream := StreamRegistry.GetStream(streamID)
		stream.Pause(capacity)
	}
}

// ResumeAllStreams sends the Resume() command to all streams this producer is
// listening to.
func (prod *ProducerBase) ResumeAllStreams() {
	for _, streamID := range prod.streams {
		stream := StreamRegistry.GetStream(streamID)
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

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.Drop(msg)
		return // ### return, closing down ###
	}

	switch msg.Enqueue(prod.messages, usedTimeout) {
	case MessageStateTimeout:
		prod.Drop(msg)
		prod.setState(PluginStateWaiting)

	case MessageStateDiscard:
		CountDiscardedMessage()
		prod.setState(PluginStateWaiting)

	default:
		prod.setState(PluginStateActive)
	}
}

// Drop routes the message to the configured drop stream.
func (prod *ProducerBase) Drop(msg Message) {
	CountDroppedMessage()

	//Log.Debug.Print("Dropping message from ", StreamRegistry.GetStreamName(msg.StreamID))
	msg.Source = prod
	msg.Route(prod.dropStreamID)
}

// CloseMessageChannel closes and empties the internal channel with a given
// timeout per message. If flushing messages runs into this timeout all
// remaining messages will be dropped by using the producer.Drop function.
// If a timout has been detected, false is returned.
func (prod *ProducerBase) CloseMessageChannel(handleMessage func(msg Message)) bool {
	close(prod.messages)
	flushWorker := new(shared.WaitGroup)

	for msg := range prod.messages {
		// handleMessage may block. To be able to exit this method we need to
		// call it async and wait for it to finish.
		flushWorker.Inc()
		go func() {
			defer flushWorker.Done()
			handleMessage(msg)
		}()

		if !flushWorker.WaitFor(prod.shutdownTimeout) {
			Log.Warning.Printf("A producer listening to %s has found to be blocking during close. Dropping remaining messages.", StreamRegistry.GetStreamName(prod.Streams()[0]))
			for msg := range prod.messages {
				prod.Drop(msg)
			}
			return false // ### return, timed out ###
		}
	}

	return true
}

func (prod *ProducerBase) tickerLoop(interval time.Duration, onTimeOut func()) {
	flushTicker := time.NewTicker(interval)
	defer flushTicker.Stop()
	for prod.IsActive() {
		<-flushTicker.C
		onTimeOut()
	}
}

func (prod *ProducerBase) messageLoop(onMessage func(Message)) {
	for prod.IsActive() {
		msg := <-prod.messages
		onMessage(msg)
	}
}

// WaitForDependencies waits until all dependencies reach the given runstate.
// A timeout > 0 can be given to work around possible blocking situations.
func (prod *ProducerBase) WaitForDependencies(waitForState PluginState, timeout time.Duration) {
	spinner := shared.NewSpinner(shared.SpinPriorityMedium)
	for _, dep := range prod.dependencies {
		start := time.Now()
		for dep.GetState() < waitForState {
			spinner.Yield()
			if timeout > 0 && time.Since(start) > timeout {
				Log.Warning.Printf("WaitForDependencies call timed out for %T", dep)
				break // ### break loop, timeout ###
			}
		}
	}
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messags. Upon stop control message doExit will be set to true.
func (prod *ProducerBase) ControlLoop() {
	prod.setState(PluginStateActive)
	defer prod.setState(PluginStateDead)

	for {
		command := <-prod.control
		switch command {
		default:
			Log.Debug.Print("Recieved untracked command")
			// Do nothing

		case PluginControlStopProducer, PluginControlStop:
			Log.Debug.Print("Recieved stop command")
			prod.WaitForDependencies(PluginStateDead, prod.GetShutdownTimeout())
			prod.setState(PluginStateStopping)
			Log.Debug.Print("All dependencies resolved")

			if prod.onStop != nil {
				prod.onStop()
			}
			return // ### return ###

		case PluginControlRoll:
			Log.Debug.Print("Recieved roll command")
			if prod.onRoll != nil {
				prod.onRoll()
			}
		}
	}
}

// MessageControlLoop provides a producer mainloop that is sufficient for most
// usecases. ControlLoop will be called in a separate go routine.
// This function will block until a stop signal is recieved.
func (prod *ProducerBase) MessageControlLoop(onMessage func(Message)) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	prod.messageLoop(onMessage)
}

// TickerMessageControlLoop is like MessageLoop but executes a given function at every
// given interval tick, too.
func (prod *ProducerBase) TickerMessageControlLoop(onMessage func(Message), interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	go prod.tickerLoop(interval, onTimeOut)
	prod.messageLoop(onMessage)
}
