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

package distributor

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"math/rand"
)

// Random distributor plugin
// Configuration example
//
//   - "distributor.Random":
//     Enable: true
//     Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to a random producer in the set of the producers listening
// to the given stream.
type Random struct {
	core.DistributorBase
}

func init() {
	shared.RuntimeType.Register(Random{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *Random) Configure(conf core.PluginConfig) error {
	return nil
}

// Distribute sends the given message to one random producer in the set of
// given producers.
func (dist *Random) Distribute(msg core.Message) {
	index := rand.Intn(len(dist.DistributorBase.Producers))
	dist.DistributorBase.Producers[index].Post(msg)
}
