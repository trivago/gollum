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

// Broadcast stream plugin
// Messages will be sent to all producers attached to this router.
type Broadcast struct {
	core.SimpleRouter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Broadcast) Configure(conf core.PluginConfigReader) error {
	return router.SimpleRouter.Configure(conf)
}

// Start the router
func (router *Broadcast) Start() error {
	return nil
}

// Enqueue enques a message to the router
func (router *Broadcast) Enqueue(msg *core.Message) error {
	producers := router.GetProducers()
	numProducers := len(producers)

	switch numProducers {
	case 0:
		return core.NewModulateResultError("No producers configured for stream %s", router.GetID())

	case 1:
		producers[0].Enqueue(msg, router.GetTimeout())

	default:
		lastProdIdx := numProducers - 1
		for prodIdx := 0; prodIdx < lastProdIdx; prodIdx++ {
			producers[prodIdx].Enqueue(msg.Clone(), router.GetTimeout())
		}
		producers[lastProdIdx].Enqueue(msg, router.GetTimeout())
	}

	return nil
}
