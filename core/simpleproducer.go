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
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/thealthcheck"
)

// SimpleProducer producer
//
// This type defines a common baseclass for all producers. All producer plugins
// should derive from this class as all required basic functions are already
// implemented here in a general way.
//
// Parameters
//
// - Streams: Defines a list of streams the producer will receive from. This
// parameter is mandatory. Specifying "*" causes the producer to receive messages
// from all streams except internal internal ones (e.g. _GOLLUM_).
// By default this parameter is set to an empty list.
//
// - FallbackStream: Defines a stream to route messages to if delivery fails.
// The message is reset to its original state before being routed, i.e. all
// modifications done to the message after leaving the consumer are removed.
// Setting this paramater to "" will cause messages to be discared when delivery
// fails.
//
// - ShutdownTimeoutMs: Defines the maximum time in milliseconds a producer is
// allowed to take to shut down. After this timeout the producer is always
// considered to have shut down.  Decreasing this value may lead to lost
// messages during shutdown. Raising it may increase shutdown time.
//
// - Modulators: Defines a list of modulators to be applied to a message when
// it arrives at this producer. If a modulator changes the stream of a message
// the message is NOT routed to this stream anymore.
// By default this parameter is set to an empty list.
type SimpleProducer struct {
	id              string
	control         chan PluginControl
	runState        *PluginRunState
	streams         []MessageStreamID `config:"Streams"`
	modulators      ModulatorArray    `config:"Modulators"`
	fallbackStream  Router            `config:"FallbackStream" default:""`
	shutdownTimeout time.Duration     `config:"ShutdownTimeoutMs" default:"1000" metric:"ms"`
	onRoll          func()
	onPrepareStop   func()
	onStop          func()
	Logger          logrus.FieldLogger
}

// Configure initializes the standard producer config values.
func (prod *SimpleProducer) Configure(conf PluginConfigReader) {
	prod.id = conf.GetID()
	prod.Logger = conf.GetLogger()
	prod.runState = NewPluginRunState()
	prod.control = make(chan PluginControl, 1)

	// Simple health check for the plugin state
	//   Path: "/<plugin_id>/pluginState"
	prod.AddHealthCheckAt("/pluginState", func() (code int, body string) {
		if prod.IsActive() {
			return thealthcheck.StatusOK,
				fmt.Sprintf("ACTIVE: %s", prod.runState.GetStateString())
		}
		return thealthcheck.StatusServiceUnavailable,
			fmt.Sprintf("NOT_ACTIVE: %s", prod.runState.GetStateString())
	})
}

// GetLogger returns the logging scope of this plugin
func (prod *SimpleProducer) GetLogger() logrus.FieldLogger {
	return prod.Logger
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
	if len(prod.modulators) > 0 {
		msg.FreezeOriginal()
		return prod.modulators.Modulate(msg)
	}
	return ModulateResultContinue
}

// HasContinueAfterModulate applies all modulators by Modulate, handle the ModulateResult
// and return if you have to continue the message process.
// This method is a default producer modulate handling.
func (prod *SimpleProducer) HasContinueAfterModulate(msg *Message) bool {
	switch result := prod.Modulate(msg); result {
	case ModulateResultDiscard:
		DiscardMessage(msg, prod.GetID(), "Producer discarded")
		return false

	case ModulateResultFallback:
		if err := Route(msg, msg.GetRouter()); err != nil {
			prod.Logger.WithError(err).Error("Failed to route to fallback")
		}
		return false

	case ModulateResultContinue:
		// OK
		return true

	default:
		prod.Logger.Error("Modulator result not supported:", result)
		return false
	}
}

// TryFallback routes the message to the configured fallback stream.
func (prod *SimpleProducer) TryFallback(msg *Message) {
	if err := RouteOriginal(msg, prod.fallbackStream); err != nil {
		prod.Logger.WithError(err).Error("Failed to route to fallback")
	}
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messags. Upon stop control message doExit will be set to true.
func (prod *SimpleProducer) ControlLoop() {
	prod.setState(PluginStateActive)
	defer prod.setState(PluginStateDead)
	defer prod.Logger.Debug("Stopped")

	for {
		command := <-prod.control
		switch command {
		default:
			prod.Logger.Debug("Received untracked command")
			// Do nothing

		case PluginControlStopProducer:
			prod.Logger.Debug("Preparing for stop")
			prod.setState(PluginStatePrepareStop)

			if prod.onPrepareStop != nil {
				if !tgo.ReturnAfter(prod.shutdownTimeout*5, prod.onPrepareStop) {
					prod.Logger.Error("Timeout during onPrepareStop.")
				}
			}

			prod.Logger.Debug("Executing stop command")
			prod.setState(PluginStateStopping)

			if prod.onStop != nil {
				if !tgo.ReturnAfter(prod.shutdownTimeout*5, prod.onStop) {
					prod.Logger.Error("Timeout during onStop.")
				}
			}
			return // ### return ###

		case PluginControlRoll:
			prod.Logger.Debug("Received roll command")
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
