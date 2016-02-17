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
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tsync"
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

// ProducerBase plugin base type
// This type defines a common baseclass for producers. All producers should
// derive from this class, but not necessarily need to.
// Configuration example:
//
//  - "producer.Foobar":
//    Enable: true
//    ID: ""
//    Channel: 8192
//    ChannelTimeoutMs: 0
//    ShutdownTimeoutMs: 3000
//    Formatter: "format.Forward"
//    Filter: "filter.All"
//    DropToStream: "_DROPPED_"
//    Fuse: ""
//    FuseTimeoutSec: 5
//    Stream:
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
// Each producer decides if and when to use a Formatter.
//
// Filter sets a filter that is applied before formatting, i.e. before a message
// is send to the message queue. If a producer requires filtering after
// formatting it has to define a separate filter as the producer decides if
// and where to format.
//
// Fuse defines the name of a fuse to burn if e.g. the producer encounteres a
// lost connection. Each producer defines its own fuse breaking logic if
// necessary / applyable. Disable fuse behavior for a producer by setting an
// empty  name or a FuseTimeoutSec <= 0. By default this is set to "".
//
// FuseTimeoutSec defines the interval in seconds used to check if the fuse can
// be recovered. Note that automatic fuse recovery logic depends on each
// producer's implementation. By default this setting is set to 10.
type ProducerBase struct {
	messages        chan Message
	control         chan PluginControl
	streams         []MessageStreamID
	dropStreamID    MessageStreamID
	dependencies    []Producer
	runState        *PluginRunState
	timeout         time.Duration
	shutdownTimeout time.Duration
	fuseTimeout     time.Duration
	fuseControl     *time.Timer
	fuseName        string
	formatters      []Formatter
	filters         []Filter
	onRoll          func()
	onStop          func()
	onCheckFuse     func() bool
	Log             tlog.LogScope
}

// Configure initializes the standard producer config values.
func (prod *ProducerBase) Configure(conf PluginConfigReader) error {
	prod.Log = conf.GetLogScope()
	prod.runState = NewPluginRunState()

	formatPlugins := conf.GetPluginArray("Formatters", []Plugin{})

	for _, plugin := range formatPlugins {
		formatter, isFormatter := plugin.(Formatter)
		if !isFormatter {
			conf.Errors.Pushf("Plugin is not a valid formatter")
		} else {
			formatter.SetLogScope(prod.Log)
			prod.formatters = append(prod.formatters, formatter)
		}
	}

	filterPlugins := conf.GetPluginArray("Filters", []Plugin{})

	for _, plugin := range filterPlugins {
		filter, isFilter := plugin.(Filter)
		if !isFilter {
			conf.Errors.Pushf("Plugin is not a valid filter")
		} else {
			filter.SetLogScope(prod.Log)
			prod.filters = append(prod.filters, filter)
		}
	}

	prod.control = make(chan PluginControl, 1)
	prod.streams = conf.GetStreamArray("Streams", []MessageStreamID{WildcardStreamID})
	prod.messages = make(chan Message, conf.GetInt("Channel", 8192))
	prod.timeout = time.Duration(conf.GetInt("ChannelTimeoutMs", 0)) * time.Millisecond
	prod.shutdownTimeout = time.Duration(conf.GetInt("ShutdownTimeoutMs", 3000)) * time.Millisecond
	prod.fuseTimeout = time.Duration(conf.GetInt("FuseTimeoutSec", 10)) * time.Second
	prod.fuseName = conf.GetString("Fuse", "")
	prod.dropStreamID = GetStreamID(conf.GetString("DropToStream", DroppedStream))

	prod.onRoll = nil
	prod.onStop = nil

	return conf.Errors.OrNil()
}

// setState sets the runstate of this plugin
func (prod *ProducerBase) setState(state PluginState) {
	if state == prod.GetState() {
		return // ### return, no change ###
	}

	prod.runState.SetState(state)
}

// GetFuse returns the fuse bound to this producer or nil if no fuse name has
// been set.
func (prod *ProducerBase) GetFuse() *tsync.Fuse {
	if prod.fuseName == "" || prod.fuseTimeout <= 0 {
		return nil
	}
	return StreamRegistry.GetFuse(prod.fuseName)
}

// GetState returns the state this plugin is currently in
func (prod *ProducerBase) GetState() PluginState {
	return prod.runState.GetState()
}

// IsBlocked returns true if GetState() returns waiting
func (prod *ProducerBase) IsBlocked() bool {
	return prod.GetState() == PluginStateWaiting
}

// IsActive returns true if GetState() returns active or waiting
func (prod *ProducerBase) IsActive() bool {
	return prod.GetState() <= PluginStateActive
}

// IsStopping returns true if GetState() returns stopping
func (prod *ProducerBase) IsStopping() bool {
	return prod.GetState() == PluginStateStopping
}

// IsActiveOrStopping is a shortcut for prod.IsActive() || prod.IsStopping()
func (prod *ProducerBase) IsActiveOrStopping() bool {
	return prod.IsActive() || prod.IsStopping()
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (prod *ProducerBase) SetRollCallback(onRoll func()) {
	prod.onRoll = onRoll
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (prod *ProducerBase) SetStopCallback(onStop func()) {
	prod.onStop = onStop
}

// SetCheckFuseCallback sets the function to be called upon PluginControlCheckFuse.
// The callback has to return true to trigger a fuse reactivation.
// If nil is passed as a callback PluginControlCheckFuse will reactivate the
// fuse immediately.
func (prod *ProducerBase) SetCheckFuseCallback(onCheckFuse func() bool) {
	prod.onCheckFuse = onCheckFuse
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
// Returns false if no message was received.
func (prod *ProducerBase) NextNonBlocking(onMessage func(msg Message)) bool {
	select {
	case msg := <-prod.messages:
		onMessage(msg)
		return true
	default:
		return false
	}
}

// Format calls all formatters in their order of definition
func (prod *ProducerBase) Format(msg Message) ([]byte, MessageStreamID) {
	for _, formatter := range prod.formatters {
		msg.Data, msg.StreamID = formatter.Format(msg)
	}
	return msg.Data, msg.StreamID
}

// AppendFormatter adds a given formatter to the end of the list of formatters
func (prod *ProducerBase) AppendFormatter(format Formatter) {
	prod.formatters = append(prod.formatters, format)
}

// PrependFormatter adds a given formatter to the start of the list of formatters
func (prod *ProducerBase) PrependFormatter(format Formatter) {
	prod.formatters = append([]Formatter{format}, prod.formatters...)
}

// Accepts calls the filters Accepts function
func (prod *ProducerBase) Accepts(msg Message) bool {
	for _, filter := range prod.filters {
		if !filter.Accepts(msg) {
			return false // ### return, false if one filter failed ###
		}
	}
	return true
}

// AppendFilter adds a given filter to the end of the list of filters
func (prod *ProducerBase) AppendFilter(filter Filter) {
	prod.filters = append(prod.filters, filter)
}

// PrependFilter adds a given filter to the start of the list of filters
func (prod *ProducerBase) PrependFilter(filter Filter) {
	prod.filters = append([]Filter{filter}, prod.filters...)
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
	defer func() {
		if r := recover(); r != nil {
			prod.Log.Error.Print("Recovered a panic during producer enqueue: ", r)
			prod.Log.Error.Print("State: ", prod.GetState(), ", Stream: ", StreamRegistry.GetStreamName(msg.StreamID))
			prod.Drop(msg)
		}
	}()

	// Filtering happens before formatting. If fitering AFTER formatting is
	// required, the producer has to do so as it decides where to format.
	if !prod.Accepts(msg) {
		CountFilteredMessage()
		return // ### return, filtered ###
	}

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.Drop(msg)
		return // ### return, closing down ###
	}

	// Allow timeout overwrite
	usedTimeout := prod.timeout
	if timeout != nil {
		usedTimeout = *timeout
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
	flushWorker := new(tsync.WaitGroup)

	for msg := range prod.messages {
		// handleMessage may block. To be able to exit this method we need to
		// call it async and wait for it to finish.
		flushWorker.Inc()
		go func() {
			defer flushWorker.Done()
			handleMessage(msg)
		}()

		if !flushWorker.WaitFor(prod.shutdownTimeout) {
			prod.Log.Warning.Printf("A producer listening to %s has found to be blocking during close. Dropping remaining messages.", StreamRegistry.GetStreamName(prod.Streams()[0]))
			for msg := range prod.messages {
				prod.Drop(msg)
			}
			return false // ### return, timed out ###
		}
	}

	return true
}

func (prod *ProducerBase) tickerLoop(interval time.Duration, onTimeOut func()) {
	if prod.IsActive() {
		start := time.Now()
		onTimeOut()

		// Delay the next call so that interval is approximated. If the timeout
		// call took longer than expected, the next function will be called
		// immediately.
		nextDelay := interval - time.Since(start)
		if nextDelay < 0 {
			go prod.tickerLoop(interval, onTimeOut)
		} else {
			time.AfterFunc(nextDelay, func() { prod.tickerLoop(interval, onTimeOut) })
		}
	}
}

func (prod *ProducerBase) messageLoop(onMessage func(Message)) {
	for prod.IsActive() {
		msg, more := <-prod.messages
		if more {
			onMessage(msg)
		}
	}
}

// WaitForDependencies waits until all dependencies reach the given runstate.
// A timeout > 0 can be given to work around possible blocking situations.
func (prod *ProducerBase) WaitForDependencies(waitForState PluginState, timeout time.Duration) {
	spinner := tsync.NewSpinner(tsync.SpinPriorityMedium)
	for _, dep := range prod.dependencies {
		start := time.Now()
		for dep.GetState() < waitForState {
			spinner.Yield()
			if timeout > 0 && time.Since(start) > timeout {
				prod.Log.Warning.Printf("WaitForDependencies call timed out for %T", dep)
				break // ### break loop, timeout ###
			}
		}
	}
}

func (prod *ProducerBase) triggerCheckFuse() {
	if fuse := prod.GetFuse(); prod.onCheckFuse != nil && fuse.IsBurned() {
		if prod.onCheckFuse() {
			prod.Control() <- PluginControlFuseActive
		} else {
			prod.fuseControl = time.AfterFunc(prod.fuseTimeout, prod.triggerCheckFuse)
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
			prod.Log.Debug.Print("Recieved untracked command")
			// Do nothing

		case PluginControlStopProducer:
			prod.Log.Debug.Print("Recieved stop command")
			prod.WaitForDependencies(PluginStateDead, prod.GetShutdownTimeout())
			prod.setState(PluginStateStopping)
			prod.Log.Debug.Print("All dependencies resolved")

			if prod.onStop != nil {
				prod.onStop()
			}
			return // ### return ###

		case PluginControlRoll:
			prod.Log.Debug.Print("Recieved roll command")
			if prod.onRoll != nil {
				prod.onRoll()
			}

		case PluginControlFuseBurn:
			if fuse := prod.GetFuse(); fuse != nil && !fuse.IsBurned() {
				fuse.Burn()
				go prod.triggerCheckFuse()
				prod.Log.Note.Print("Fuse burned")
			}

		case PluginControlFuseActive:
			if fuse := prod.GetFuse(); fuse != nil && fuse.IsBurned() {
				if prod.fuseControl != nil {
					prod.fuseControl.Stop()
				}
				fuse.Activate()
				prod.Log.Note.Print("Fuse reactivated")
			}
		}
	}
}

// MessageControlLoop provides a producer mainloop that is sufficient for most
// usecases. ControlLoop will be called in a separate go routine.
// This function will block until a stop signal is received.
func (prod *ProducerBase) MessageControlLoop(onMessage func(Message)) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	prod.messageLoop(onMessage)
}

// TickerMessageControlLoop is like MessageLoop but executes a given function at
// every given interval tick, too. If the onTick function takes longer than
// interval, the next tick will be delayed until onTick finishes.
func (prod *ProducerBase) TickerMessageControlLoop(onMessage func(Message), interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	go prod.tickerLoop(interval, onTimeOut)
	prod.messageLoop(onMessage)
}
