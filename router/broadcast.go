// Copyright 2015-2018 trivago N.V.
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

// Broadcast router
//
// This router implements the default behavior of routing all messages to all
// producers registered to the configured stream.
//
// Examples
//
//  rateLimiter:
//    Type: router.Broadcast
//    Stream: errorlogs
//    Filters:
//      - filter.Rate:
//        MessagesPerSec: 200
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
	if len(producers) == 0 {
		return core.NewModulateResultError(
			"Router %s: no producers configured", router.GetID())
	}

	timeout := router.GetTimeout()
	lastProdIdx := len(producers) - 1
	for _, prod := range producers[:lastProdIdx] {
		prod.Enqueue(msg.Clone(), timeout)
	}

	// Cloning is a rather expensive operation, so skip cloning for the last
	// message (not required)
	producers[lastProdIdx].Enqueue(msg, timeout)
	return nil
}
