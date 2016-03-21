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

package librdkafka

import "sync/atomic"

// Partitioner is the interface used for partitioner callbacks passed to
// librdkafka.
type Partitioner interface {
	GetPartition(key []byte, numPartitions int) int
}

// RoundRobinPartitioner is a default implementation of the Partitioner
// interface. This partitioner will iterate all partitions in a round
// robin manner.
type RoundRobinPartitioner struct {
	counter *uint64
}

// NewRoundRobinPartitioner creates a new round robin partitioner.
func NewRoundRobinPartitioner() RoundRobinPartitioner {
	return RoundRobinPartitioner{
		counter: new(uint64),
	}
}

// GetPartition returns the next partition as in (p+1) % n.
// This method is threadsafe and lockfree.
func (r RoundRobinPartitioner) GetPartition(key []byte, numPartitions int) int {
	idx := int(atomic.AddUint64(r.counter, 1) - 1)
	return idx % numPartitions
}
