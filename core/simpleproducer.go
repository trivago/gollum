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
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tsync"
	"sync"
	"time"
)

// SimpleProducer plugin base type
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
// a blocking producer during shutdown. By default this is set to 1 second.
// Decreasing this value may lead to lost messages during shutdown. Increasing
// this value will increase shutdown time.
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
type SimpleProducer struct {
	id               string
	control          chan PluginControl
	streams          []MessageStreamID
	formatters       []Formatter
	filters          []Filter
	dropStreamID     MessageStreamID
	runState         *PluginRunState
	shutdownTimeout  time.Duration
	fuseTimeout      time.Duration
	fuseControlGuard *sync.Mutex
	fuseControl      *time.Timer
	fuseName         string
	onRoll           func()
	onPrepareStop    func()
	onStop           func()
	onCheckFuse      func() bool
	Log              tlog.LogScope
}

// Configure initializes the standard producer config values.
func (prod *SimpleProducer) Configure(conf PluginConfigReader) error {
	prod.id = conf.GetID()
	prod.Log = conf.GetLogScope()
	prod.runState = NewPluginRunState()
	prod.control = make(chan PluginControl, 1)
	prod.streams = conf.GetStreamArray("Streams", []MessageStreamID{WildcardStreamID})
	prod.shutdownTimeout = time.Duration(conf.GetInt("ShutdownTimeoutMs", 1000)) * time.Millisecond
	prod.fuseTimeout = time.Duration(conf.GetInt("FuseTimeoutSec", 10)) * time.Second
	prod.fuseName = conf.GetString("Fuse", "")
	prod.dropStreamID = StreamRegistry.GetStreamID(conf.GetString("DropToStream", DroppedStream))
	prod.fuseControlGuard = new(sync.Mutex)

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

	return conf.Errors.OrNil()
}

// GetID returns the ID of this producer
func (prod *SimpleProducer) GetID() string {
	return prod.id
}

// Streams returns the streams this producer is listening to.
func (prod *SimpleProducer) Streams() []MessageStreamID {
	return prod.streams
}

// GetDropStreamID returns the id of the stream to drop messages to.
func (prod *SimpleProducer) GetDropStreamID() MessageStreamID {
	return prod.dropStreamID
}

// Control returns write access to this producer's control channel.
// See PluginControl* constants.
func (prod *SimpleProducer) Control() chan<- PluginControl {
	return prod.control
}

// GetFuse returns the fuse bound to this producer or nil if no fuse name has
// been set.
func (prod *SimpleProducer) GetFuse() *tsync.Fuse {
	if prod.fuseName == "" || prod.fuseTimeout <= 0 {
		return nil
	}
	return StreamRegistry.GetFuse(prod.fuseName)
}

// GetState returns the state this plugin is currently in
func (prod *SimpleProducer) GetState() PluginState {
	return prod.runState.GetState()
}

// IsBlocked returns true if GetState() returns waiting
func (prod *SimpleProducer) IsBlocked() bool {
	return prod.GetState() == PluginStateWaiting
}

// IsActive returns true if GetState() returns initializing, active, waiting, or
// prepareStop
func (prod *SimpleProducer) IsActive() bool {
	return prod.GetState() <= PluginStatePrepareStop
}

// IsStopping returns true if GetState() returns prepareStop, stopping or dead
func (prod *SimpleProducer) IsStopping() bool {
	return prod.GetState() >= PluginStatePrepareStop
}

// IsActiveOrStopping is a shortcut for prod.IsActive() || prod.IsStopping()
func (prod *SimpleProducer) IsActiveOrStopping() bool {
	return prod.IsActive() || prod.IsStopping()
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (prod *SimpleProducer) SetRollCallback(onRoll func()) {
	prod.onRoll = onRoll
}

// SetPrepareStopCallback sets the function to be called upon PluginControlPrepareStop
func (prod *SimpleProducer) SetPrepareStopCallback(onPrepareStop func()) {
	prod.onPrepareStop = onPrepareStop
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (prod *SimpleProducer) SetStopCallback(onStop func()) {
	prod.onStop = onStop
}

// SetCheckFuseCallback sets the function to be called upon PluginControlCheckFuse.
// The callback has to return true to trigger a fuse reactivation.
// If nil is passed as a callback PluginControlCheckFuse will reactivate the
// fuse immediately.
func (prod *SimpleProducer) SetCheckFuseCallback(onCheckFuse func() bool) {
	prod.onCheckFuse = onCheckFuse
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (prod *SimpleProducer) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	prod.runState.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (prod *SimpleProducer) AddMainWorker(workers *sync.WaitGroup) {
	prod.runState.SetWorkerWaitGroup(workers)
	prod.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (prod *SimpleProducer) AddWorker() {
	prod.runState.AddWorker()
}

// WorkerDone removes an additional worker to the waitgroup.
func (prod *SimpleProducer) WorkerDone() {
	prod.runState.WorkerDone()
}

// GetShutdownTimeout returns the duration gollum will wait for this producer
// before canceling the shutdown process.
func (prod *SimpleProducer) GetShutdownTimeout() time.Duration {
	return prod.shutdownTimeout
}

// Format calls all formatters in their order of definition
func (prod *SimpleProducer) Format(msg *Message) {
	for _, formatter := range prod.formatters {
		formatter.Format(msg)
	}
}

// AppendFormatter adds a given formatter to the end of the list of formatters
func (prod *SimpleProducer) AppendFormatter(format Formatter) {
	prod.formatters = append(prod.formatters, format)
}

// PrependFormatter adds a given formatter to the start of the list of formatters
func (prod *SimpleProducer) PrependFormatter(format Formatter) {
	prod.formatters = append([]Formatter{format}, prod.formatters...)
}

// Accepts calls the filters Accepts function
func (prod *SimpleProducer) Accepts(msg *Message) bool {
	for _, filter := range prod.filters {
		if !filter.Accepts(msg) {
			filter.Drop(msg)
			return false // ### return, false if one filter failed ###
		}
	}
	return true
}

// AppendFilter adds a given filter to the end of the list of filters
func (prod *SimpleProducer) AppendFilter(filter Filter) {
	prod.filters = append(prod.filters, filter)
}

// PrependFilter adds a given filter to the start of the list of filters
func (prod *SimpleProducer) PrependFilter(filter Filter) {
	prod.filters = append([]Filter{filter}, prod.filters...)
}

// Drop routes the message to the configured drop stream.
func (prod *SimpleProducer) Drop(msg *Message) {
	CountDroppedMessage()

	//Log.Debug.Print("Dropping message from ", StreamRegistry.GetStreamName(msg.StreamID))
	msg.Source = prod
	msg.Route(prod.dropStreamID)
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messags. Upon stop control message doExit will be set to true.
func (prod *SimpleProducer) ControlLoop() {
	prod.setState(PluginStateActive)
	defer prod.setState(PluginStateDead)
	defer prod.Log.Debug.Print("Stopped")

	for {
		command := <-prod.control
		switch command {
		default:
			prod.Log.Debug.Print("Recieved untracked command")
			// Do nothing

		case PluginControlStopProducer:
			prod.Log.Debug.Print("Preparing for stop")
			prod.setState(PluginStatePrepareStop)

			if prod.onPrepareStop != nil {
				if !tgo.ReturnAfter(prod.shutdownTimeout*5, prod.onPrepareStop) {
					prod.Log.Error.Print("Timeout during onPrepareStop.")
				}
			}

			prod.Log.Debug.Print("Executing stop command")
			prod.setState(PluginStateStopping)

			if prod.onStop != nil {
				if !tgo.ReturnAfter(prod.shutdownTimeout*5, prod.onStop) {
					prod.Log.Error.Print("Timeout during onStop.")
				}
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
				prod.setFuseControl(nil)
				fuse.Activate()
				prod.Log.Note.Print("Fuse reactivated")
			}
		}
	}
}

// TickerControlLoop is like ControlLoop but executes a given function at
// every given interval tick, too. If the onTick function takes longer than
// interval, the next tick will be delayed until onTick finishes.
func (prod *SimpleProducer) TickerControlLoop(interval time.Duration, onTimeOut func()) {
	prod.setState(PluginStateActive)
	go prod.ControlLoop()
	prod.tickerLoop(interval, onTimeOut)
}

func (prod *SimpleProducer) setState(state PluginState) {
	if state != prod.GetState() {
		prod.runState.SetState(state)
	}
}

func (prod *SimpleProducer) tickerLoop(interval time.Duration, onTimeOut func()) {
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

func (prod *SimpleProducer) setFuseControl(callback func()) {
	prod.fuseControlGuard.Lock()
	defer prod.fuseControlGuard.Unlock()

	if prod.fuseControl != nil {
		prod.fuseControl.Stop()
	}

	if callback == nil {
		prod.fuseControl = nil
	} else {
		prod.fuseControl = time.AfterFunc(prod.fuseTimeout, callback)
	}
}

func (prod *SimpleProducer) triggerCheckFuse() {
	if fuse := prod.GetFuse(); prod.onCheckFuse != nil && fuse.IsBurned() {
		if prod.onCheckFuse() {
			prod.Control() <- PluginControlFuseActive
		} else {
			prod.setFuseControl(prod.triggerCheckFuse)
		}
	}
}
