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
// Configuration example:
//
// # Generate junk
// JunkGenerator:
//   Type: "consumer.Profiler"
//   Message: "%20s"
//   Streams: "junkstream"
//   Characters: "abcdefghijklmZ"
//   KeepRunning: true
//   Runs: 10000
//   Batches: 3000000
//   DelayMs: 500
// # Filter messages and distribute them to 2 new streams
// JunkRouterDist:
//   Type: "router.Distribute"
//   Stream: "junkstream"
//   TargetStreams:
//     - "junkdist_00"
//     - "junkdist_01"
//   Filters:
//     - JunkRegexp:
//         Type: "filter.RegExp"
//         Expression: "Z"
// # Print messages from junkdist_00
// JunkDistPrinter00:
//   Type: "producer.Console"
//   Streams: "junkdist_00"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junk_00] "
// # Print messages from junkdist_01
// JunkDistPrinter01:
//   Type: "producer.Console"
//   Streams: "junkdist_01"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junk_01] "
//
// TargetStreams Specifies names of the streams to create.
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
	numStreams := len(router.routers)

	switch numStreams {
	case 0:
		return core.NewModulateResultError("No producers configured for stream %s", router.GetID())

	case 1:
		router.route(msg, router.routers[0])

	default:
		lastStreamIdx := numStreams - 1
		for streamIdx := 0; streamIdx < lastStreamIdx; streamIdx++ {
			router.route(msg.Clone(), router.routers[streamIdx])
			router.Logger.Debugf("routed to StreamID '%v'", router.routers[streamIdx].GetStreamID())
		}
		router.route(msg, router.routers[lastStreamIdx])
		router.Logger.Debugf("routed to StreamID '%v'", router.routers[lastStreamIdx].GetStreamID())
	}

	return nil
}
