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
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/treflect"
	"sync"
	"sync/atomic"
)

// TypeRegistry is the global typeRegistry instance.
// Use this instance to register plugins.
var TypeRegistry = treflect.NewTypeRegistry()

var metricActiveWorkers = "ActiveWorkers"

// PluginControl is an enumeration used to pass signals to plugins
type PluginControl int

// PluginState is an enumeration used to describe the current working state of a plugin
type PluginState int32

const (
	// PluginControlStopProducer will cause any producer to halt and shutdown.
	PluginControlStopProducer = PluginControl(iota)
	// PluginControlStopConsumer will cause any consumer to halt and shutdown.
	PluginControlStopConsumer = PluginControl(iota)
	// PluginControlRoll notifies the consumer/producer about a reconnect or reopen request
	PluginControlRoll = PluginControl(iota)
	// PluginControlFuseBurn notfies a producer to burn a fuse or a consumer that a fuse has been burned
	PluginControlFuseBurn = PluginControl(iota)
	// PluginControlFuseActive notfies a producer to activate a fuse or a consumer that a fuse has been activated
	PluginControlFuseActive = PluginControl(iota)
)

const (
	// -------------------------------------------------------------------------
	// The order of these constants is important and need to reflect the typical
	// lifecycle of a plugin orderd by time.
	// -------------------------------------------------------------------------

	// PluginStateWaiting is set when a plugin is active but currently unable to process data
	PluginStateWaiting = PluginState(iota)
	// PluginStateActive is set when a plugin is ready to process data
	PluginStateActive = PluginState(iota)
	// PluginStateStopping is set when a plugin is about to stop
	PluginStateStopping = PluginState(iota)
	// PluginStateDead is set when a plugin is unable to process any data
	PluginStateDead = PluginState(iota)
)

// PluginRunState is used in some plugins to store information about the
// execution state of the plugin (i.e. if it is running or not) as well as
// threading primitives that enable gollum to wait for a plugin top properly
// shut down.
type PluginRunState struct {
	workers *sync.WaitGroup
	state   int32 // Pluginstate
}

// Plugin is the base class for any runtime class that can be configured and
// instantiated during runtim.
type Plugin interface {
	// Configure is called during NewPluginWithType
	Configure(conf PluginConfig) error
}

// PluginWithState allows certain plugins to give information about their runstate
type PluginWithState interface {
	Plugin
	GetState() PluginState
}

func init() {
	tgo.Metric.New(metricActiveWorkers)
}

// NewPluginLogScope creates a new tlog.LogScope for the plugin contained in this
// config.
func NewPluginLogScope(conf PluginConfig) tlog.LogScope {
	if conf.ID == "" {
		return tlog.NewLogScope(conf.Typename)
	}
	return tlog.NewLogScope(conf.Typename + ":" + conf.ID)
}

// NewSubPluginLogScope creates a new sub scope tlog.LogScope for the plugin contained
// in this config.
func NewSubPluginLogScope(conf PluginConfig, subScope string) tlog.LogScope {
	if conf.ID == "" {
		return tlog.NewLogScope(conf.Typename + ":" + subScope)
	}
	return tlog.NewLogScope(conf.Typename + ":" + conf.ID + ":" + subScope)
}

// NewPluginRunState creates a new plugin state helper
func NewPluginRunState() *PluginRunState {
	return &PluginRunState{
		workers: nil,
		state:   int32(PluginStateDead),
	}
}

// GetState returns the current plugin state casted to the correct type
func (state *PluginRunState) GetState() PluginState {
	return PluginState(atomic.LoadInt32(&state.state))
}

// SetState sets a new plugin state casted to the correct type
func (state *PluginRunState) SetState(nextState PluginState) {
	atomic.SwapInt32(&state.state, int32(nextState))
}

// SetWorkerWaitGroup sets the WaitGroup used to manage workers
func (state *PluginRunState) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	state.workers = workers
}

// AddWorker adds a worker to the waitgroup configured by SetWorkerWaitGroup.
func (state *PluginRunState) AddWorker() {
	state.workers.Add(1)
	tgo.Metric.Inc(metricActiveWorkers)
}

// WorkerDone removes a worker from the waitgroup configured by
// SetWorkerWaitGroup.
func (state *PluginRunState) WorkerDone() {
	state.workers.Done()
	tgo.Metric.Dec(metricActiveWorkers)
}

// NewPlugin creates a new plugin from the type information stored in its
// config. This function internally calls NewPluginWithType.
func NewPlugin(config PluginConfig) (Plugin, error) {
	obj, err := TypeRegistry.New(config.Typename)
	if err != nil {
		return nil, err
	}

	plugin, isPlugin := obj.(Plugin)
	if !isPlugin {
		return nil, fmt.Errorf("%s is not a plugin type", config.Typename)
	}

	err = plugin.Configure(config)
	if err == nil {
		if config.ID != "" {
			// If an id is set it must be unique
			PluginRegistry.RegisterUnique(plugin, config.ID)
		}
		// Check for errors (log as warning)
		if config.Validate() {
			//tlog.Debug.Printf("%#v", plugin)
		}
	}

	return plugin, err
}
