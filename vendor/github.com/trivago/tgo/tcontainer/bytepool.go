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

package tcontainer

import (
	"runtime"
	"sync"
)

// BytePool holds a set of byte buffers that are recycled after use.
// Buffers up to a size of 1000 KB can be managed by this pool, larger allocations
// will fall back to standard methods (i.e. make).
// This pool is useful for applications that need to handle a lot of arbitrary
// sized buffers. As this pool is GC backed no "put back" call is required.
type BytePool struct {
	tiny   slabsList // 64 to 960 byte
	small  slabsList // 1KB to 9KB
	medium slabsList // 10KB to 90KB
	large  slabsList // 100KB to 1000KB
}

type chunk []byte
type slab chan chunk
type slabsList struct {
	slabs     []slab
	chunkSize int
	numChunks int
	guard     *sync.Mutex
}

const (
	// tinySlabUnit is the multiplicator for tiny slabs
	tinyChunkSize = 64
	// tinySlabCount is the number of tiny slabs
	tinySlabCount = (smallChunkSize / tinyChunkSize) - 1
	// tinySlabMax is the largest tiny slab available
	tinyChunkMax = tinyChunkSize * tinySlabCount

	// smallSlabUnit is the multiplicator for small slabs
	smallChunkSize = 1024
	// smallSlabCount is the number of small slabs
	smallSlabCount = (mediumChunkSize / smallChunkSize) - 1
	// smallSlabMax is the largest small slab available
	smallChunkMax = smallChunkSize * smallSlabCount

	// mediumSlabUnit is the multiplicator for medium slabs
	mediumChunkSize = 1024 * 10
	// mediumSlabCount is the number of medium slabs
	mediumSlabCount = (largeChunkSize / mediumChunkSize) - 1
	// mediumSlabMax is the largest medium slab available
	mediumChunkMax = mediumChunkSize * mediumSlabCount

	// largeSlabUnit is the multiplicator for large slabs
	largeChunkSize = 1024 * 100
	// largeSlabCount is the number of large slabs
	largeSlabCount = 10
	// largeSlabMax is the largest large slab available
	largeChunkMax = largeChunkSize * largeSlabCount
)

// NewBytePool creates a new bytepool. This will call NewBytePoolWithSize
// with 1000, 100, 100, 10.
func NewBytePool() BytePool {
	return NewBytePoolWithSize(1000, 100, 100, 10)
}

// NewBytePoolWithSize creates a new BytePool. You can define the maximum
// number of chunks ([]byte) cached per storage size. Chunks wills always be
// returned, even if the cache is empty. If there are more chunks "in flight"
// than allowed, excess chunks will be garbage collected.
// Storage sizes in bytes: tiny: 1-960, small: 1024-9216, medium: 10240-92160.
// large: 102400-1024000. Each sorage size holds up to 10 slabs.
func NewBytePoolWithSize(tinyChunkCount int, smallChunkCount int, mediumChunkCount int, largeChunkCount int) BytePool {
	return BytePool{
		tiny:   newSlabsList(tinySlabCount, tinyChunkCount, tinyChunkSize),
		small:  newSlabsList(smallSlabCount, smallChunkCount, smallChunkSize),
		medium: newSlabsList(mediumSlabCount, mediumChunkCount, mediumChunkSize),
		large:  newSlabsList(largeSlabCount, largeChunkCount, largeChunkSize),
	}
}

// Get returns a byte slice that can hold the given number of bytes.
// If possible this slice is coming from a pool. Slices are automatically
// returned to the pool. No additional action is necessary.
func (b *BytePool) Get(size int) []byte {
	if size == 0 {
		return []byte{}
	}

	slab := b.getSlab(size)
	if slab == nil {
		return make([]byte, size) // ### return, oversized ###
	}

	select {
	case buffer := <-slab:
		return buffer[:size] // ### return, cached ###

	default:
		return make([]byte, size) // ### return, empty pool ###
	}
}

func newSlabsList(numSlabs int, numChunks int, chunkSize int) slabsList {
	return slabsList{
		slabs:     make([]slab, numSlabs),
		chunkSize: chunkSize,
		numChunks: numChunks,
		guard:     new(sync.Mutex),
	}
}

func (b *BytePool) getSlab(size int) slab {
	switch {

	case size <= tinyChunkMax:
		return b.tiny.fetch(size, b)

	case size <= smallChunkMax:
		return b.small.fetch(size, b)

	case size <= mediumChunkMax:
		return b.medium.fetch(size, b)

	case size <= largeChunkMax:
		return b.large.fetch(size, b)

	default:
		return nil // ### return, too large ###
	}
}

func (s *slabsList) fetch(size int, b *BytePool) slab {
	slabIdx := size / (s.chunkSize + 1)
	chunks := s.slabs[slabIdx]

	if chunks == nil {
		// First initialization can be racey
		s.guard.Lock()
		defer s.guard.Unlock()

		if chunks = s.slabs[slabIdx]; chunks == nil {
			chunks = make(slab, s.numChunks)
			chunkSize := (slabIdx + 1) * s.chunkSize

			// Prepopulate slab
			for i := 0; i < s.numChunks; i++ {
				buffer := make([]byte, chunkSize)
				runtime.SetFinalizer(&buffer, b.put)
				chunks <- buffer
			}

			s.slabs[slabIdx] = chunks
		}
	}

	return chunks
}

func (b *BytePool) put(buffer *[]byte) {
	slab := b.getSlab(cap(*buffer))
	select {
	case slab <- *buffer:
		runtime.SetFinalizer(buffer, b.put)
	default:
		// discard, pool is full
	}
}
