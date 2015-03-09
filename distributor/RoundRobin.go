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
	"github.com/trivago/gollum/shared"
)

// RoundRobin distributor plugin
// Configuration example
//
//   - "distributor.RoundRobin":
//     Enable: true
//     Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to one of the producers listening to the given stream.
// The target producer changes after each send.
type RoundRobin struct {
	shared.DistributorBase
	index map[shared.MessageStreamID]int
}

func init() {
	shared.RuntimeType.Register(RoundRobin{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *RoundRobin) Configure(conf shared.PluginConfig) error {
	dist.index = make(map[shared.MessageStreamID]int)
	return nil
}

// Distribute sends the given message to one of the given producers in a round
// robin fashion.
func (dist *RoundRobin) Distribute(msg shared.Message) {

	// As we might listen to different streams we have to keep the index for
	// each stream separately
	index, isSet := dist.index[msg.CurrentStream]
	if !isSet {
		index = 0
	} else {
		index %= len(dist.DistributorBase.Producers)
	}

	msg.SendTo(dist.DistributorBase.Producers[index])
	dist.index[msg.CurrentStream] = index + 1
}
