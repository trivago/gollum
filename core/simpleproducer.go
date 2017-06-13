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
	"fmt"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/thealthcheck"
	"github.com/trivago/tgo/tlog"
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
// 	  Modulators:
//    	- "filter.All"
//    Formatter: "format.Forward"
//    FallbackStream: ""
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
// A timeout of -1 or lower will try the fallback route without notice.
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
// FallbackStream defines the stream used for messages that cannot be delivered
// e.g. after a timeout (see ChannelTimeoutMs). By default this is "".
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
type SimpleProducer struct {
	id              string
	control         chan PluginControl
	runState        *PluginRunState
	streams         []MessageStreamID `config:"Streams" default:"*"`
	modulators      ModulatorArray    `config:"Modulators"`
	fallbackStream  Router            `config:"FallbackStream" default:""`
	shutdownTimeout time.Duration     `config:"ShutdownTimeoutMs" default:"1000" metric:"ms"`
	onRoll          func()
	onPrepareStop   func()
	onStop          func()
	Log             tlog.LogScope
}

// Configure initializes the standard producer config values.
func (prod *SimpleProducer) Configure(conf PluginConfigReader) error {
	prod.id = conf.GetID()
	prod.Log = conf.GetLogScope()
	prod.runState = NewPluginRunState()
	prod.control = make(chan PluginControl, 1)
	//prod.streams = conf.GetStreamArray("Streams", []MessageStreamID{WildcardStreamID})
	//prod.shutdownTimeout = time.Duration(conf.GetInt("ShutdownTimeoutMs", 1000)) * time.Millisecond
	//prod.modulators = conf.GetModulatorArray("Modulators", prod.Log, ModulatorArray{})

	//fallbackStreamID := StreamRegistry.GetStreamID(conf.GetString("FallbackStream", InvalidStream))
	//prod.fallbackStream = StreamRegistry.GetRouterOrFallback(fallbackStreamID)

	// Simple health check for the plugin state
	//   Path: "/<plugin_id>/SimpleProducer/pluginstate"
	prod.AddHealthCheckAt("/pluginState", func() (code int, body string) {
		if prod.IsActive() {
			return thealthcheck.StatusOK, fmt.Sprintf("ACTIVE: %s", prod.GetStateString())
		}
		return thealthcheck.StatusServiceUnavailable,
			fmt.Sprintf("NOT_ACTIVE: %s", prod.GetStateString())
	})
	return conf.Errors.OrNil()
}

// GetLogScope returns the logging scope of this plugin
func (prod *SimpleProducer) GetLogScope() tlog.LogScope {
	return prod.Log
}

// AddHealthCheck adds a health check at the default URL (http://<addr>:<port>/<plugin_id>)
func (prod *SimpleProducer) AddHealthCheck(callback thealthcheck.CallbackFunc) {
	prod.AddHealthCheckAt("", callback)
}

// AddHealthCheckAt adds a health check at a subpath (http://<addr>:<port>/<plugin_id><path>)
func (prod *SimpleProducer) AddHealthCheckAt(path string, callback thealthcheck.CallbackFunc) {
	thealthcheck.AddEndpoint("/"+prod.GetID()+path, callback)
}

// GetID returns the ID of this producer
func (prod *SimpleProducer) GetID() string {
	return prod.id
}

// Streams returns the routers this producer is listening to.
func (prod *SimpleProducer) Streams() []MessageStreamID {
	return prod.streams
}

// Control returns write access to this producer's control channel.
// See PluginControl* constants.
func (prod *SimpleProducer) Control() chan<- PluginControl {
	return prod.control
}

// GetState returns the state this plugin is currently in
func (prod *SimpleProducer) GetState() PluginState {
	return prod.runState.GetState()
}

// GetStateString returns the name of state this plugin is currently in
func (prod *SimpleProducer) GetStateString() string {
	return stateToDescription[prod.runState.GetState()]
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

// Modulate applies all modulators from this producer to a given message.
// This implementation handles routing and discarding of messages.
func (prod *SimpleProducer) Modulate(msg *Message) ModulateResult {
	return prod.modulators.Modulate(msg)
}

// HasContinueAfterModulate applies all modulators by Modulate, handle the ModulateResult
// and return if you have to continue the message process.
// This method is a default producer modulate handling.
func (prod *SimpleProducer) HasContinueAfterModulate(msg *Message) bool {
	switch result := prod.Modulate(msg); result {
	case ModulateResultDiscard:
		DiscardMessage(msg)
		return false

	case ModulateResultFallback:
		RouteOriginal(msg, msg.GetRouter())
		return false

	case ModulateResultContinue:
		// OK
		return true

	default:
		prod.Log.Error.Print("Modulator result not supported:", result)
		return false
	}
}

// TryFallback routes the message to the configured fallback stream.
func (prod *SimpleProducer) TryFallback(msg *Message) {
	RouteOriginal(msg, prod.fallbackStream)
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
			prod.Log.Debug.Print("Received untracked command")
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
			prod.Log.Debug.Print("Received roll command")
			if prod.onRoll != nil {
				prod.onRoll()
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
