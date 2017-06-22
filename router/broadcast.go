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

// Broadcast router plugin
//
// The "Broadcast" plugin relays all messages sent to [Stream]  to all
// produces connected to [Stream].
//
// With the default options, this plugin has no practical effect. Using
// the Filters and TimeoutMs options, it is possible to restrict and control
// the messages being passed.
//
// Configuration example:
//
// # Generate junk messages
// JunkGenerator:
//   Type: "consumer.Profiler"
//   Message: "%20s"
//   Streams: "junkstream"
//   Characters: "abcdefghijklmZ"
//   KeepRunning: true
//   Runs: 10000
//   Batches: 3000000
//   DelayMs: 500
// # Filter out messsages including 'Z' and broadcast them
// JunkRouterBroad:
//   Type: "router.Broadcast"
//   Stream: "junkstream"
//   Filters:
//     - JunkRegexp:
//         Type: "filter.RegExp"
//         Expression: "Z"
// # Produce filtered messages from 'junkstream' to stdout
// JunkPrinter00:
//   Type: "producer.Console"
//   Streams: "junkstream"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junkdist_00] "
// # Produce filtered messages from 'junkstream' to stdout
// JunkPrinter01:
//   Type: "producer.Console"
//   Streams: "junkstream"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junkdist_01] "
//
type Broadcast struct {
	core.SimpleRouter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Broadcast) Configure(conf core.PluginConfigReader) {
}

// Start the router
func (router *Broadcast) Start() error {
	return nil
}

// Enqueue enques a message to the router
func (router *Broadcast) Enqueue(msg *core.Message) error {
	producers := router.GetProducers()

	switch len(producers) {
	default:
		for prodIdx := 1; prodIdx < len(producers); prodIdx++ {
			producers[prodIdx].Enqueue(msg.Clone(), router.GetTimeout())
		}
		fallthrough

	case 1:
		producers[0].Enqueue(msg, router.GetTimeout())
		return nil

	case 0:
		return core.NewModulateResultError(
			"Router %s: no producers configured", router.GetID())
	}
}
