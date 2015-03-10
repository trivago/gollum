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

// Broadcast distributor plugin
// Configuration example
//
//   - "distributor.Standard":
//     Enable: true
//     Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to all of the producers listening to the given stream.
type Broadcast struct {
	shared.DistributorBase
}

func init() {
	shared.RuntimeType.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *Broadcast) Configure(conf shared.PluginConfig) error {
	return nil
}

// Distribute sends the given message to all of the given producers
func (dist *Broadcast) Distribute(msg shared.Message) {
	for _, prod := range dist.DistributorBase.Producers {
		prod.Post(msg)
	}
}
