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

// Random stream plugin
// Messages will be sent to one of the producers attached to this router.
// The concrete producer is chosen randomly with each message.
type Random struct {
	core.SimpleRouter `gollumdoc:embed_type`
}

func init() {
	core.TypeRegistry.Register(Random{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Random) Configure(conf core.PluginConfigReader) error {
	return router.SimpleRouter.Configure(conf)
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
	producers[index].Enqueue(msg, router.Timeout)
	return nil
}
