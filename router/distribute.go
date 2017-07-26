// Copyright 2015-2017 trivago GmbH
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

package router

import (
	"github.com/trivago/gollum/core"
)

// Distribute router plugin
//
// The "Distribute" plugin provides 1:n stream remapping by duplicating
// messages.
//
// During startup, it creates a set of streams with names listed
// in [TargetStreams]. During execution, it consumes messages from
// the stream [Stream] and enqueues copies of these messages onto
// each of the streams listed in [TargetStreams].
//
// When routing to multiple routers, the incoming stream has to be listed
// explicitly to be used.
//
// Parameters
//
// - TargetStreams: List of streams to route the incoming messages to.
//
// Examples
//
// This example route incoming messages from `streamA` to `streamB` and `streamC` (duplication):
//
//  JunkRouterDist:
//    Type: router.Distribute
//    Stream: streamA
//    TargetStreams:
//      - streamB
//      - streamC
//
type Distribute struct {
	Broadcast      `gollumdoc:"embed_type"`
	routers        []core.Router
	boundStreamIDs []core.MessageStreamID
}

func init() {
	core.TypeRegistry.Register(Distribute{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Distribute) Configure(conf core.PluginConfigReader) {
	router.boundStreamIDs = conf.GetStreamArray("TargetStreams", []core.MessageStreamID{})
}

// Start the router
func (router *Distribute) Start() error {
	for _, streamID := range router.boundStreamIDs {
		targetRouter := core.StreamRegistry.GetRouterOrFallback(streamID)
		router.routers = append(router.routers, targetRouter)
	}
	return nil
}

func (router *Distribute) route(msg *core.Message, targetRouter core.Router) {
	if router.GetStreamID() == targetRouter.GetStreamID() {
		router.Broadcast.Enqueue(msg)
	} else {
		msg.SetStreamID(targetRouter.GetStreamID())
		core.Route(msg, targetRouter)
	}
}

// Enqueue enques a message to the router
func (router *Distribute) Enqueue(msg *core.Message) error {
	routers := router.routers
	if len(routers) == 0 {
		return core.NewModulateResultError(
			"Router %s: no streams configured", router.GetID())
	}

	lastRouterIdx := len(routers) - 1
	for _, targetRouter := range routers[:lastRouterIdx] {
		router.route(msg.Clone(), targetRouter)
	}

	// Cloning is a rather expensive operation, so skip cloning for the last
	// message (not required)
	router.route(msg, routers[lastRouterIdx])
	return nil
}
