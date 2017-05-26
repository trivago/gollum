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
	"github.com/trivago/tgo/thealthcheck"
	"github.com/trivago/tgo/tlog"
	"sync"
	"time"
)

// SimpleConsumer plugin base type
// This type defines a common baseclass for all consumers. All consumer plugins
// should derive from this class but don't necessarily need to.
// Configuration example:
//
//  - "consumer.Foobar":
//    Enable: true
//    ID: ""
//    ShutdownTimeoutMs: 1000
//    Router:
//      - "foo"
//      - "bar"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// ID allows this consumer to be found by other plugins by name. By default this
// is set to "" which does not register this consumer.
//
// Router contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all routers" will get these
// messages.
//
// ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect
// various timeouts during shutdown. By default this is set to 1 second.
type SimpleConsumer struct {
	id              string
	control         chan PluginControl
	runState        *PluginRunState
	routers         []Router       `config:"Streams"`
	shutdownTimeout time.Duration  `config:"ShutdownTimeoutMs" default:"1000" metric:"ms"`
	modulators      ModulatorArray `config:"Modulators"`
	onRoll          func()
	onPrepareStop   func()
	onStop          func()
	Log             tlog.LogScope
}

// Configure initializes standard consumer values from a plugin config.
func (cons *SimpleConsumer) Configure(conf PluginConfigReader) error {
	conf.Configure(cons, cons.Log)

	cons.id = conf.GetID()
	cons.Log = conf.GetLogScope()
	cons.runState = NewPluginRunState()
	cons.control = make(chan PluginControl, 1)
	//cons.modulators = conf.GetModulatorArray("Modulators", cons.Log, ModulatorArray{})

	//defaultStreamID := GetStreamID(conf.GetID())
	//boundStreamIDs := conf.GetStreamArray("Streams", []MessageStreamID{defaultStreamID})

	//for _, streamID := range boundStreamIDs {
	//	stream := StreamRegistry.GetRouterOrFallback(streamID)
	//	cons.routers = append(cons.routers, stream)
	//}

	//cons.shutdownTimeout = time.Duration(conf.GetInt("ShutdownTimeoutMs", 1000)) * time.Millisecond

	return conf.Errors.OrNil()
}

// AddHealthCheck a health check at the default URL
// (http://<addr>:<port>/<plugin_id>)
func (cons *SimpleConsumer) AddHealthCheck(callback thealthcheck.CallbackFunc) {
	cons.AddHealthCheckAt("", callback)
}

// AddHealthCheckAt adds a health check at a subpath
// (http://<addr>:<port>/<plugin_id><path>)
func (cons *SimpleConsumer) AddHealthCheckAt(path string, callback thealthcheck.CallbackFunc) {
	thealthcheck.AddEndpoint("/"+cons.GetID()+path, callback)
}

// GetID returns the ID of this consumer
func (cons *SimpleConsumer) GetID() string {
	return cons.id
}

// GetShutdownTimeout returns the duration gollum will wait for this producer
// before canceling the shutdown process.
func (cons *SimpleConsumer) GetShutdownTimeout() time.Duration {
	return cons.shutdownTimeout
}

// Control returns write access to this consumer's control channel.
// See ConsumerControl* constants.
func (cons *SimpleConsumer) Control() chan<- PluginControl {
	return cons.control
}

// GetState returns the state this plugin is currently in
func (cons *SimpleConsumer) GetState() PluginState {
	return cons.runState.GetState()
}

// IsBlocked returns true if GetState() returns waiting
func (cons *SimpleConsumer) IsBlocked() bool {
	return cons.GetState() == PluginStateWaiting
}

// IsActive returns true if GetState() returns initialize, active, waiting or
// prepareStop.
func (cons *SimpleConsumer) IsActive() bool {
	return cons.GetState() <= PluginStatePrepareStop
}

// IsStopping returns true if GetState() returns prepareStop, stopping or dead
func (cons *SimpleConsumer) IsStopping() bool {
	return cons.GetState() >= PluginStatePrepareStop
}

// IsActiveOrStopping is a shortcut for prod.IsActive() || prod.IsStopping()
func (cons *SimpleConsumer) IsActiveOrStopping() bool {
	return cons.IsActive() || cons.IsStopping()
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (cons *SimpleConsumer) SetRollCallback(onRoll func()) {
	cons.onRoll = onRoll
}

// SetPrepareStopCallback sets the function to be called upon PluginControlPrepareStop
func (cons *SimpleConsumer) SetPrepareStopCallback(onPrepareStop func()) {
	cons.onPrepareStop = onPrepareStop
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (cons *SimpleConsumer) SetStopCallback(onStop func()) {
	cons.onStop = onStop
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (cons *SimpleConsumer) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	cons.runState.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (cons *SimpleConsumer) AddMainWorker(workers *sync.WaitGroup) {
	cons.runState.SetWorkerWaitGroup(workers)
	cons.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (cons *SimpleConsumer) AddWorker() {
	cons.runState.AddWorker()
}

// WorkerDone removes an additional worker to the waitgroup.
func (cons *SimpleConsumer) WorkerDone() {
	cons.runState.WorkerDone()
}

// Enqueue creates a new message from a given byte slice and passes it to
// EnqueueMessage. Data is copied to the message.
func (cons *SimpleConsumer) Enqueue(data []byte) {
	msg := NewMessage(cons, data, GetStreamID(cons.id))
	cons.enqueueMessage(msg)
}

// EnqueueWithMetadata works like EnqueueWithSequence and allows to set meta data directly
func (cons *SimpleConsumer) EnqueueWithMetadata(data []byte, metaData Metadata) {
	msg := NewMessage(cons, data, GetStreamID(cons.id))

	msg.data.Metadata = metaData
	msg.orig.Metadata = metaData.Clone()
	cons.enqueueMessage(msg)
}

func (cons *SimpleConsumer) enqueueMessage(msg *Message) {
	// Execute configured modulators
	switch cons.modulators.Modulate(msg) {
	case ModulateResultDiscard:
		DiscardMessage(msg)
		return

	case ModulateResultFallback:
		if err := RouteOriginal(msg, msg.GetRouter()); err != nil {
			cons.Log.Error.Print(err)
		}
		return
	}

	// Send message to all routers registered to this consumer
	// Last message will not be cloned.
	numRouters := len(cons.routers)
	lastStreamIdx := numRouters - 1

	for streamIdx := 0; streamIdx < lastStreamIdx; streamIdx++ {
		router := cons.routers[streamIdx]
		msg := msg.Clone()
		msg.SetStreamID(router.GetStreamID())

		if err := Route(msg, router); err != nil {
			cons.Log.Error.Print(err)
		}
	}

	router := cons.routers[lastStreamIdx]
	msg.SetStreamID(router.GetStreamID())

	if err := Route(msg, router); err != nil {
		cons.Log.Error.Print(err)
	}
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messages. Upon stop control message doExit will be set to true.
func (cons *SimpleConsumer) ControlLoop() {
	cons.setState(PluginStateActive)
	defer cons.setState(PluginStateDead)
	defer cons.Log.Debug.Print("Stopped")

	for {
		command := <-cons.control
		switch command {
		default:
			cons.Log.Debug.Print("Received untracked command")
			// Do nothing

		case PluginControlStopConsumer:
			cons.Log.Debug.Print("Preparing for stop")
			cons.setState(PluginStatePrepareStop)

			if cons.onPrepareStop != nil {
				if !tgo.ReturnAfter(cons.shutdownTimeout*5, cons.onPrepareStop) {
					cons.Log.Error.Print("Timeout during onPrepareStop")
				}
			}

			cons.Log.Debug.Print("Executing stop command")
			cons.setState(PluginStateStopping)

			if cons.onStop != nil {
				if !tgo.ReturnAfter(cons.shutdownTimeout*5, cons.onStop) {
					cons.Log.Error.Printf("Timeout during onStop")
				}
			}
			return // ### return ###

		case PluginControlRoll:
			cons.Log.Debug.Print("Received roll command")
			if cons.onRoll != nil {
				cons.onRoll()
			}
		}
	}
}

// TickerControlLoop is like MessageLoop but executes a given function at
// every given interval tick, too. Note that the interval is not exact. If the
// onTick function takes longer than interval, the next tick will be delayed
// until onTick finishes.
func (cons *SimpleConsumer) TickerControlLoop(interval time.Duration, onTick func()) {
	cons.setState(PluginStateActive)
	go cons.tickerLoop(interval, onTick)
	cons.ControlLoop()
}

func (cons *SimpleConsumer) setState(state PluginState) {
	cons.runState.SetState(state)
}

func (cons *SimpleConsumer) tickerLoop(interval time.Duration, onTimeOut func()) {
	if cons.IsActive() {
		start := time.Now()
		onTimeOut()

		// Delay the next call so that interval is approximated. If the timeout
		// call took longer than expected, the next function will be called
		// immediately.
		nextDelay := interval - time.Since(start)
		if nextDelay < 0 {
			go cons.tickerLoop(interval, onTimeOut)
		} else {
			time.AfterFunc(nextDelay, func() { cons.tickerLoop(interval, onTimeOut) })
		}
	}
}
