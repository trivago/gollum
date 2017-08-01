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
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/thealthcheck"
	"sync"
	"time"
)

// SimpleConsumer consumer
//
// This type defines a common baseclass for all consumers. All consumer plugins
// should derive from this class as all required basic functions are already
// implemented in a general way.
//
// Parameters
//
// - Streams: Defines a list of streams a consumer will send to. This parameter
// is mandatory. When using "*" messages will be sent only to the internal "*"
// stream. It will NOT send messages to all streams.
// By default this parameter is set to an empty list.
//
// - ShutdownTimeoutMs: Defines the maximum time in milliseconds a consumer is
// allowed to take to shut down. After this timeout the consumer is always
// considered to have shut down.
// By default this parameter is set to 1000.
//
// - Modulators: Defines a list of modulators to be applied to a message before
// it is sent to the list of streams. If a modulator specifies a stream, the
// message is only sent to that specific stream. A message is saved as original
// after all modulators have been applied.
// By default this parameter is set to an empty list.
//
// - ModulatorRoutines: Defines the number of go routines reserved for
// modulating messages. Setting this parameter to 0 will use as many go routines
// as the specific consumer plugin is using for fetching data. Any other value
// will force the given number fo go routines to be used.
// By default this parameter is set to 0
//
// - ModulatorQueueSize: Defines the size of the channel used to buffer messages
// before they are fetched by the next free modulator go routine. If the
// ModulatorRoutines parameter is set to 0 this parameter is ignored.
// By default this parameter is set to 1024.
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
	enqueueMessage  func(*Message)
	modulatorQueue  MessageQueue
	Logger          logrus.FieldLogger
}

// Configure initializes standard consumer values from a plugin config.
func (cons *SimpleConsumer) Configure(conf PluginConfigReader) {
	cons.id = conf.GetID()
	cons.Logger = conf.GetLogger()
	cons.runState = NewPluginRunState()
	cons.control = make(chan PluginControl, 1)

	numRoutines := conf.GetInt("ModulatorRoutines", 0)
	queueSize := conf.GetInt("ModulatorQueueSize", 1024)

	if numRoutines > 0 {
		cons.Logger.Debugf("Using %d modulator routines", numRoutines)
		cons.modulatorQueue = NewMessageQueue(int(queueSize))
		for i := 0; i < int(numRoutines); i++ {
			go cons.processQueue()
		}
		cons.enqueueMessage = cons.parallelEnqueue
	} else {
		cons.enqueueMessage = cons.directEnqueue
	}
}

// GetLogger returns the logger scoped to this plugin
func (cons *SimpleConsumer) GetLogger() logrus.FieldLogger {
	return cons.Logger
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
	cons.EnqueueWithMetadata(data, nil)
}

// EnqueueWithMetadata works like EnqueueWithSequence and allows to set meta data directly
func (cons *SimpleConsumer) EnqueueWithMetadata(data []byte, metaData Metadata) {
	msg := NewMessage(cons, data, metaData, InvalidStreamID)
	cons.enqueueMessage(msg)
}

func (cons *SimpleConsumer) parallelEnqueue(msg *Message) {
	cons.modulatorQueue.Push(msg, 0)
}

func (cons *SimpleConsumer) processQueue() {
loop:
	if msg, hasMore := cons.modulatorQueue.Pop(); hasMore {
		cons.directEnqueue(msg)
		goto loop
	}
}

func (cons *SimpleConsumer) directEnqueue(msg *Message) {
	// Execute configured modulators
	switch cons.modulators.Modulate(msg) {
	case ModulateResultDiscard:
		DiscardMessage(msg)
		return

	case ModulateResultFallback:
		if err := RouteOriginal(msg, msg.GetRouter()); err != nil {
			cons.Logger.Error(err)
		}
		return
	}

	CountMessagesEnqueued()

	// Send message to all routers registered to this consumer
	// Last message will not be cloned.
	numRouters := len(cons.routers)
	lastStreamIdx := numRouters - 1

	for streamIdx := 0; streamIdx < lastStreamIdx; streamIdx++ {
		router := cons.routers[streamIdx]
		msg := msg.Clone()
		msg.SetStreamID(router.GetStreamID())
		msg.FreezeOriginal()

		if err := Route(msg, router); err != nil {
			cons.Logger.Error(err)
		}
	}

	router := cons.routers[lastStreamIdx]
	msg.SetStreamID(router.GetStreamID())
	msg.FreezeOriginal()

	if err := Route(msg, router); err != nil {
		cons.Logger.Error(err)
	}
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messages. Upon stop control message doExit will be set to true.
func (cons *SimpleConsumer) ControlLoop() {
	cons.setState(PluginStateActive)
	defer cons.setState(PluginStateDead)
	defer cons.Logger.Debug("Stopped")

	for {
		command := <-cons.control
		switch command {
		default:
			cons.Logger.Debug("Received untracked command")
			// Do nothing

		case PluginControlStopConsumer:
			cons.Logger.Debug("Preparing for stop")
			cons.setState(PluginStatePrepareStop)

			if cons.onPrepareStop != nil {
				if !tgo.ReturnAfter(cons.shutdownTimeout*5, cons.onPrepareStop) {
					cons.Logger.Error("Timeout during onPrepareStop")
				}
			}

			cons.Logger.Debug("Executing stop command")
			cons.setState(PluginStateStopping)

			if cons.onStop != nil {
				if !tgo.ReturnAfter(cons.shutdownTimeout*5, cons.onStop) {
					cons.Logger.Errorf("Timeout during onStop")
				}
			}

			if cons.modulatorQueue != nil {
				close(cons.modulatorQueue)
			}
			return // ### return ###

		case PluginControlRoll:
			cons.Logger.Debug("Received roll command")
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
