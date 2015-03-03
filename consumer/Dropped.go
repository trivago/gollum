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

package consumer

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Dropped consumer plugin
// Configuration example
//
//   - "consumer.Dropped":
//     Enable: true
//
// This consumer does not define any options beside the standard ones.
type Dropped struct {
	shared.ConsumerBase
}

func init() {
	shared.RuntimeType.Register(Dropped{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Dropped) Configure(conf shared.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}
	//cons.ConsumerBase.streams = []shared.MessageStreamID{shared.DroppedStreamID}
	shared.RegisterDropConsumer(cons.ConsumerBase)
	return nil
}

// Consume is not doing anything as this consumer is just a host for a channel.
func (cons Dropped) Consume(threads *sync.WaitGroup) {
}
