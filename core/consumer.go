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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/tgo"
	"sync"
	"time"
)

// Consumer is an interface for plugins that receive data from outside sources
// and generate Message objects from this data.
type Consumer interface {
	PluginWithState
	MessageSource

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
//   ID: ""
//   Fuse: ""
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// ID allows this consumer to be found by other plugins by name. By default this
// is set to "" which does not register this consumer.
//
// Stream contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all streams" will get these
// messages.
//
// Fuse defines the name of a fuse to observe for this consumer. Producer may
// "burn" the fuse when they encounter errors. Consumers may react on this by
// e.g. closing connections to notify any writing services of the problem.
// Set to "" by default which disables the fuse feature for this consumer.
// It is up to the consumer implementation to react on a broken fuse in an
// appropriate manner.
type ConsumerBase struct {
	control      chan PluginControl
	streams      []MappedStream
	runState     *PluginRunState
	fuse         *tgo.Fuse
	onRoll       func()
	onStop       func()
	onFuseBurned func()
	onFuseActive func()
}

// Configure initializes standard consumer values from a plugin config.
func (cons *ConsumerBase) Configure(conf PluginConfig) error {
	cons.runState = NewPluginRunState()
	cons.control = make(chan PluginControl, 1)
	cons.onRoll = nil
	cons.onStop = nil

	for _, streamName := range conf.Stream {
		streamID := GetStreamID(streamName)
		cons.streams = append(cons.streams, MappedStream{
			StreamID: streamID,
			Stream:   StreamRegistry.GetStreamOrFallback(streamID),
		})
	}

	fuseName := conf.GetString("Fuse", "")
	if fuseName != "" {
		cons.fuse = StreamRegistry.GetFuse(fuseName)
	}

	return nil
}

// setState sets the runstate of this plugin
func (cons *ConsumerBase) setState(state PluginState) {
	cons.runState.SetState(state)
}

// GetState returns the state this plugin is currently in
func (cons *ConsumerBase) GetState() PluginState {
	return cons.runState.GetState()
}

// WaitOnFuse blocks if the fuse linked to this consumer has been burned.
// If no fuse is bound this function does nothing.
func (cons *ConsumerBase) WaitOnFuse() {
	if cons.fuse != nil {
		cons.fuse.Wait()
	}
}

// IsFuseBurned returns true if the fuse linked to this consumer has been
// burned. If no fuse is attached, false is returned.
func (cons *ConsumerBase) IsFuseBurned() bool {
	if cons.fuse == nil {
		return false
	}
	return cons.fuse.IsBurned()
}

// IsBlocked returns true if GetState() returns waiting
func (cons *ConsumerBase) IsBlocked() bool {
	return cons.GetState() == PluginStateWaiting
}

// IsActive returns true if GetState() returns active
func (cons *ConsumerBase) IsActive() bool {
	return cons.GetState() <= PluginStateActive
}

// IsStopping returns true if GetState() returns stopping
func (cons *ConsumerBase) IsStopping() bool {
	return cons.GetState() == PluginStateStopping
}

// IsActiveOrStopping is a shortcut for prod.IsActive() || prod.IsStopping()
func (cons *ConsumerBase) IsActiveOrStopping() bool {
	return cons.IsActive() || cons.IsStopping()
}

// SetRollCallback sets the function to be called upon PluginControlRoll
func (cons *ConsumerBase) SetRollCallback(onRoll func()) {
	cons.onRoll = onRoll
}

// SetStopCallback sets the function to be called upon PluginControlStop
func (cons *ConsumerBase) SetStopCallback(onStop func()) {
	cons.onStop = onStop
}

// SetFuseBurnedCallback sets the function to be called upon PluginControlFuseBurned
func (cons *ConsumerBase) SetFuseBurnedCallback(onFuseBurned func()) {
	cons.onFuseBurned = onFuseBurned
}

// SetFuseActiveCallback sets the function to be called upon PluginControlFuseActive
func (cons *ConsumerBase) SetFuseActiveCallback(onFuseActive func()) {
	cons.onFuseActive = onFuseActive
}

// SetWorkerWaitGroup forwards to Plugin.SetWorkerWaitGroup for this consumer's
// internal plugin state. This method is also called by AddMainWorker.
func (cons *ConsumerBase) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	cons.runState.SetWorkerWaitGroup(workers)
}

// AddMainWorker adds the first worker to the waitgroup
func (cons *ConsumerBase) AddMainWorker(workers *sync.WaitGroup) {
	cons.runState.SetWorkerWaitGroup(workers)
	cons.AddWorker()
}

// AddWorker adds an additional worker to the waitgroup. Assumes that either
// MarkAsActive or SetWaitGroup has been called beforehand.
func (cons *ConsumerBase) AddWorker() {
	cons.runState.AddWorker()
}

// WorkerDone removes an additional worker to the waitgroup.
func (cons *ConsumerBase) WorkerDone() {
	cons.runState.WorkerDone()
}

// Enqueue creates a new message from a given byte slice and passes it to
// EnqueueMessage. Note that data is not copied, just referenced by the message.
func (cons *ConsumerBase) Enqueue(data []byte, sequence uint64) {
	cons.EnqueueMessage(NewMessage(cons, data, sequence))
}

// EnqueueCopy behaves like Enqueue but creates a copy of data that is attached
// to the message.
func (cons *ConsumerBase) EnqueueCopy(data []byte, sequence uint64) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	cons.EnqueueMessage(NewMessage(cons, dataCopy, sequence))
}

// EnqueueMessage passes a given message  to all streams.
// Only the StreamID of the message is modified, everything else is passed as-is.
func (cons *ConsumerBase) EnqueueMessage(msg Message) {
	for _, mapping := range cons.streams {
		msg.StreamID = mapping.StreamID
		msg.PrevStreamID = msg.StreamID
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

func (cons *ConsumerBase) tickerLoop(interval time.Duration, onTimeOut func()) {
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

func (cons *ConsumerBase) fuseControlLoop() {
	if cons.fuse == nil {
		return // ### return, no fuse attached ###
	}
	spin := tgo.NewSpinner(tgo.SpinPrioritySuspend)
	for cons.IsActive() {
		// If the fuse is burned: callback, wait, callback
		if cons.IsFuseBurned() {
			cons.Control() <- PluginControlFuseBurn
			cons.WaitOnFuse()
			cons.Control() <- PluginControlFuseActive
		} else {
			spin.Yield()
		}
	}
}

// ControlLoop listens to the control channel and triggers callbacks for these
// messags. Upon stop control message doExit will be set to true.
func (cons *ConsumerBase) ControlLoop() {
	cons.setState(PluginStateActive)
	defer cons.setState(PluginStateDead)
	go cons.fuseControlLoop()

	for {
		command := <-cons.control
		switch command {
		default:
			Log.Debug.Print("Recieved untracked command")
			// Do nothing

		case PluginControlStopConsumer:
			cons.setState(PluginStateStopping)
			Log.Debug.Print("Recieved stop command")
			if cons.onStop != nil {
				cons.onStop()
			}
			return // ### return ###

		case PluginControlRoll:
			Log.Debug.Print("Recieved roll command")
			if cons.onRoll != nil {
				cons.onRoll()
			}

		case PluginControlFuseBurn:
			Log.Debug.Print("Recieved fuse burned command")
			if cons.onFuseBurned != nil {
				cons.onFuseBurned()
			}

		case PluginControlFuseActive:
			Log.Debug.Print("Recieved fuse active command")
			if cons.onFuseActive != nil {
				cons.onFuseActive()
			}
		}
	}
}

// TickerControlLoop is like MessageLoop but executes a given function at
// every given interval tick, too. Note that the interval is not exact. If the
// onTick function takes longer than interval, the next tick will be delayed
// until onTick finishes.
func (cons *ConsumerBase) TickerControlLoop(interval time.Duration, onTick func()) {
	cons.setState(PluginStateActive)
	go cons.tickerLoop(interval, onTick)
	cons.ControlLoop()
}
