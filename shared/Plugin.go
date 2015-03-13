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

package shared

import (
	"log"
	"os"
	"runtime/debug"
	"sync"
)

var metricActiveWorkers = "ActiveWorkers"

// PluginRunState is used in some plugins to store information about the
// execution state of the plugin (i.e. if it is running or not) as well as
// threading primitives that enable gollum to wait for a plugin top properly
// shut down.
type PluginRunState struct {
	workers *sync.WaitGroup
	paused  bool
}

// Plugin is the base class for any runtime class that can be configured and
// instantiated during runtim.
type Plugin interface {
	Configure(conf PluginConfig) error
}

func init() {
	Metric.New(metricActiveWorkers)
}

// Pause implements the MessageSource interface
func (state *PluginRunState) Pause() {
	state.paused = true
}

// IsPaused implements the MessageSource interface
func (state *PluginRunState) IsPaused() bool {
	return state.paused
}

// Resume implements the MessageSource interface
func (state *PluginRunState) Resume() {
	state.paused = false
}

// SetWorkerWaitGroup sets the WaitGroup used to manage workers
func (state *PluginRunState) SetWorkerWaitGroup(workers *sync.WaitGroup) {
	state.workers = workers
}

// AddWorker adds a worker to the waitgroup configured by SetWorkerWaitGroup.
func (state *PluginRunState) AddWorker() {
	state.workers.Add(1)
	Metric.Inc(metricActiveWorkers)
}

// WorkerDone removes a worker from the waitgroup configured by
// SetWorkerWaitGroup.
func (state *PluginRunState) WorkerDone() {
	state.workers.Done()
	Metric.Dec(metricActiveWorkers)
}

// RecoverShutdown will trigger a shutdown via interrupt if a panic was issued.
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Println(r)
		log.Println(string(debug.Stack()))

		// Send interrupt = clean shutdown
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}
