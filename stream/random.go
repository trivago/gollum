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
	"math/rand"
)

// Random stream plugin
// Configuration example
//
//   - "stream.Random":
//     Enable: true
//     Stream: "data"
//	   Formatter: "format.Envelope"
//     Filter: "filter.All"
//     StickyStream: true
//
// Messages will be sent to one of the producers attached to this stream.
// The producer used is defined randomly.
//
// This stream defines the same fields as stream.Broadcast.
type Random struct {
	core.StreamBase
}

func init() {
	shared.RuntimeType.Register(Random{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Random) Configure(conf core.PluginConfig) error {
	return stream.StreamBase.ConfigureStream(conf, stream.random)
}

func (stream *Random) random(msg core.Message) {
	index := rand.Intn(len(stream.StreamBase.Producers))
	stream.StreamBase.Producers[index].Enqueue(msg, stream.Timeout)
}
