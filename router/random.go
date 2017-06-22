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
	"math/rand"
)

// Random router plugin
//
// The "Random" router relays each message sent to the stream [Stream] to
// exactly one of the producers connected to [Stream]. The receiving producer
// is chosen randomly for each message.
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
// # Randomly assign messages to connected producers
// JunkRouterRand:
//   Type: "router.Random"
//   Stream: "junkstream"
// # Produce messages to stdout
// JunkPrinter00:
//   Type: "producer.Console"
//   Streams: "junkstream"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junk_00] "
// # Produce messages to stdout
// JunkPrinter01:
//   Type: "producer.Console"
//   Streams: "junkstream"
//   Modulators:
//     - "format.Envelope":
//         Prefix: "[junk_01] "
// 
type Random struct {
	core.SimpleRouter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Random{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Random) Configure(conf core.PluginConfigReader) {
}

// Start the router
func (router *Random) Start() error {
	return nil
}

// Enqueue enques a message to the router
func (router *Random) Enqueue(msg *core.Message) error {
	producers := router.GetProducers()
	if len(producers) == 0 {
		return core.NewModulateResultError("No producers configured for stream %s", router.GetID())
	}

	index := rand.Intn(len(producers))
	producers[index].Enqueue(msg, router.GetTimeout())
	return nil
}
