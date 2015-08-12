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
)

var metricActiveWorkers = "ActiveWorkers"

// PluginControl is an enumeration used to pass signals to plugins
type PluginControl int

// PluginState is an enumeration used to describe the current working state of a plugin
type PluginState int

const (
	// PluginControlStopProducer will cause any producer to halt and shutdown.
	PluginControlStopProducer = PluginControl(iota)
	// PluginControlStopConsumer will cause any consumer to halt and shutdown.
	PluginControlStopConsumer = PluginControl(iota)
	// PluginControlRoll notifies the consumer/producer about a reconnect or reopen request
	PluginControlRoll = PluginControl(iota)
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
	state   PluginState
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
	shared.Metric.New(metricActiveWorkers)
}

// NewPluginRunState creates a new plugin state helper
func NewPluginRunState() *PluginRunState {
	return &PluginRunState{
		workers: nil,
		state:   PluginStateDead,
	}
}

// SetWorkerWaitGroup sets the WaitGroup used to manage workers
func (state *PluginRunState) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	state.workers = workers
}

// AddWorker adds a worker to the waitgroup configured by SetWorkerWaitGroup.
func (state *PluginRunState) AddWorker() {
	state.workers.Add(1)
	shared.Metric.Inc(metricActiveWorkers)
}

// WorkerDone removes a worker from the waitgroup configured by
// SetWorkerWaitGroup.
func (state *PluginRunState) WorkerDone() {
	state.workers.Done()
	shared.Metric.Dec(metricActiveWorkers)
}

// NewPluginWithType creates a new plugin of a given type and initializes it
// using the given config (i.e. passes that config to Configure). The type
// passed to this function may differ from the type stored in the config.
// If the type is meant to match use NewPlugin instead of NewPluginWithType.
// This function returns nil, error if the plugin could not be instantiated or
// plugin, error if Configure failed.
func NewPluginWithType(typename string, config PluginConfig) (Plugin, error) {
	obj, err := shared.TypeRegistry.New(typename)
	if err != nil {
		return nil, err
	}

	plugin, isPlugin := obj.(Plugin)
	if !isPlugin {
		return nil, fmt.Errorf("%s is no plugin", typename)
	}

	err = plugin.Configure(config)

	// Nested plugins must not trigger a validation. Validation happens after
	// all plugins are configured.
	if typename == config.Typename {
		config.Validate()
	}

	// Register named plugins
	if err != nil && config.ID != "" {
		PluginRegistry.RegisterUnique(plugin, config.ID)
	}

	return plugin, err
}

// NewPlugin creates a new plugin from the type information stored in its
// config. This function internally calls NewPluginWithType.
func NewPlugin(config PluginConfig) (Plugin, error) {
	return NewPluginWithType(config.Typename, config)
}
