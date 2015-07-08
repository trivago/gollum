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
	"sync"
	"sync/atomic"
)

// RoundRobin stream plugin
// Configuration example
//
//   - "stream.RoundRobin":
//     Enable: true
//     Stream: "data"
//	   Formatter: "format.Envelope"
//     Filter: "filter.All"
//
// Messages will be sent to one of the producers attached to this stream.
// Producers will be switched one-by-one.
//
// This stream defines the same fields as stream.Broadcast.
type RoundRobin struct {
	core.StreamBase
	index         int32
	indexByStream map[core.MessageStreamID]*int32
	mapInitLock   *sync.Mutex
}

func init() {
	shared.RuntimeType.Register(RoundRobin{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *RoundRobin) Configure(conf core.PluginConfig) error {
	if err := stream.StreamBase.Configure(conf); err != nil {
		return err // ### return, base stream error ###
	}

	if stream.StickyStream {
		stream.StreamBase.Distribute = stream.roundRobinOverStream
	} else {
		stream.StreamBase.Distribute = stream.roundRobinOverAll
	}
	stream.index = 0
	stream.indexByStream = make(map[core.MessageStreamID]*int32)
	stream.mapInitLock = new(sync.Mutex)
	return nil
}

func (stream *RoundRobin) roundRobinOverAll(msg core.Message) {
	index := atomic.AddInt32(&stream.index, 1) % int32(len(stream.StreamBase.Producers))
	stream.StreamBase.Producers[index].Enqueue(msg)
}

func (stream *RoundRobin) roundRobinOverStream(msg core.Message) {
	producers, exists := stream.StreamBase.ProducersByStream[msg.StreamID]
	if !exists {
		shared.Metric.Inc(core.MetricNoRoute)
		shared.Metric.Inc(core.MetricDiscarded)
	}

	indexBase, exists := stream.indexByStream[msg.StreamID]
	if !exists {
		stream.mapInitLock.Lock()
		// Check and get (!) again -- possible race
		if indexBase, exists = stream.indexByStream[msg.StreamID]; !exists {
			indexBase = new(int32)
			stream.indexByStream[msg.StreamID] = indexBase
		}
		stream.mapInitLock.Unlock()
	}

	index := atomic.AddInt32(indexBase, 1) % int32(len(producers))
	producers[index].Enqueue(msg)
}
