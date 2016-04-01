// Copyright 2015-2016 trivago GmbH
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
	"sync"
	"sync/atomic"
)

// RoundRobin stream plugin
// Messages will be sent to one of the producers attached to this stream.
// Producers will be switched one-by-one.
type RoundRobin struct {
	core.StreamBase
	index         int32
	indexByStream map[core.MessageStreamID]*int32
	mapInitLock   *sync.Mutex
}

func init() {
	core.TypeRegistry.Register(RoundRobin{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *RoundRobin) Configure(conf core.PluginConfigReader) error {
	stream.StreamBase.ConfigureStream(conf, stream.roundRobin)

	stream.index = 0
	stream.indexByStream = make(map[core.MessageStreamID]*int32)
	stream.mapInitLock = new(sync.Mutex)
	return conf.Errors.OrNil()
}

func (stream *RoundRobin) roundRobin(msg *core.Message) {
	index := atomic.AddInt32(&stream.index, 1) % int32(len(stream.StreamBase.Producers))
	stream.StreamBase.Producers[index].Enqueue(msg, stream.Timeout)
}
