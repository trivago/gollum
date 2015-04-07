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

package stream

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"sync/atomic"
)

// RoundRobin stream plugin
// Configuration example
//
//   - "stream.RoundRobin":
//     Enable: true
//     Stream: "data"
//
// This stream does not define any options beside the standard ones.
// Messages are send to one of the producers listening to the given stream.
// The target producer changes after each send.
type RoundRobin struct {
	core.StreamBase
	index map[core.MessageStreamID]*int32
}

func init() {
	shared.RuntimeType.Register(RoundRobin{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *RoundRobin) Configure(conf core.PluginConfig) error {
	if err := stream.StreamBase.Configure(conf); err != nil {
		return err // ### return, base stream error ###
	}
	stream.index = make(map[core.MessageStreamID]*int32)
	stream.StreamBase.Distribute = stream.roundRobin
	return nil
}

func (stream *RoundRobin) roundRobin(msg core.Message) {

	// As we might listen to different streams we have to keep the index for
	// each stream separately
	if _, isSet := stream.index[msg.Stream]; !isSet {
		stream.index[msg.Stream] = new(int32)
	}

	index := atomic.AddInt32(stream.index[msg.Stream], 1)
	index = index % int32(len(stream.StreamBase.Producers))

	stream.StreamBase.Producers[index].Enqueue(msg)
}
