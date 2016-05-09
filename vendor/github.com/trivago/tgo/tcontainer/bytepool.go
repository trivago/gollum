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

// BytePool is a fragmentation friendly way to allocated byte slices.
type BytePool struct {
}

const (
	tiny   = 64
	small  = 1024
	medium = 1024 * 10
	large  = 1024 * 100
	huge   = 1024 * 1000
)

// NewBytePool creates a new BytePool
func NewBytePool() BytePool {
	return BytePool{}
}

// Get returns a slice allocated to a normalized size.
// Sizes are organized in evenly sized buckets so that fragmentation is kept low.
// Buckets are:
//  * 0      - 960     bytes (64 byte steps)
//  * 960    - 10240   bytes (1 kb steps)
//  * 10240  - 102400  bytes (10 kb steps)
//  * 102400 - 1024000 bytes (100 kb steps)
func (b *BytePool) Get(size int) []byte {
	switch {
	case size == 0:
		return []byte{}

	case size <= small-tiny:
		return make([]byte, size, ((size-1)/tiny+1)*tiny)

	case size <= medium-small:
		return make([]byte, size, ((size-1)/small+1)*small)

	case size <= large-medium:
		return make([]byte, size, ((size-1)/medium+1)*medium)

	case size <= huge-large:
		return make([]byte, size, ((size-1)/large+1)*large)

	default:
		return make([]byte, size)
	}
}

/*
import (
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
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

type slab struct {
	chunks []*uintptr
	top    *int32
	head   *int32
	size   int
}

type slabsList struct {
	slabs []slab
	unit  int
	max   int
}

const (
	// tinySlabUnit is the multiplicator for tiny slabs
	tinySlabUnit = 64
	// tinySlabCount is the number of tiny slabs
	tinySlabCount = (smallSlabUnit / tinySlabUnit) - 1

	// smallSlabUnit is the multiplicator for small slabs
	smallSlabUnit = 1024
	// smallSlabCount is the number of small slabs
	smallSlabCount = (mediumSlabUnit / smallSlabUnit) - 1

	// mediumSlabUnit is the multiplicator for medium slabs
	mediumSlabUnit = 1024 * 10
	// mediumSlabCount is the number of medium slabs
	mediumSlabCount = (largeSlabUnit / mediumSlabUnit) - 1

	// largeSlabUnit is the multiplicator for large slabs
	largeSlabUnit = 1024 * 100
	// largeSlabCount is the number of large slabs
	largeSlabCount = 10
)

// NewBytePool creates a new bytepool. This will call NewBytePoolWithSize
// with 10000, 1000, 100, 10.
func NewBytePool() BytePool {
	return BytePool{
		tiny:   newSlabsList(tinySlabCount, tinySlabUnit),
		small:  newSlabsList(smallSlabCount, smallSlabUnit),
		medium: newSlabsList(mediumSlabCount, mediumSlabUnit),
		large:  newSlabsList(largeSlabCount, largeSlabUnit),
	}
}

// Get returns a byte slice that can hold the given number of bytes.
// If possible this slice is coming from a pool. Slices are automatically
// returned to the pool. No additional action is necessary.
func (b *BytePool) Get(size int) []byte {
	switch {
	case size == 0:
		return []byte{}

	case size <= b.tiny.max:
		return b.tiny.alloc(size)

	case size <= b.small.max:
		return b.small.alloc(size)

	case size <= b.medium.max:
		return b.medium.alloc(size)

	case size <= b.large.max:
		return b.large.alloc(size)

	default:
		return make([]byte, size)
	}
}

func newSlabsList(count int, unit int) slabsList {
	slabs := make([]slab, count)
	for i := range slabs {
		slabs[i] = slab{
			chunks: make([]*uintptr, 10),
			top:    new(int32),
			head:   new(int32),
			size:   unit * (i + 1),
		}
		*slabs[i].top--
		*slabs[i].head--
	}

	return slabsList{
		slabs: slabs,
		unit:  unit,
		max:   unit * count,
	}
}

func (s *slabsList) alloc(size int) []byte {
	slabIdx := (size - 1) / s.unit
	slab := s.slabs[slabIdx]
	ptr := slab.pop()

	header := reflect.SliceHeader{
		Data: *ptr, // BROKEN: Finalizer is not moved
		Len:  size,
		Cap:  slab.size,
	}

	return *(*[]byte)(unsafe.Pointer(&header))
}

func (s *slab) push(p *uintptr) {
	for {
		top := atomic.LoadInt32(s.top)
		if atomic.CompareAndSwapInt32(s.head, top, top+1) {
			// Grow stack if necessary
			if top+1 == int32(len(s.chunks)) {
				old := s.chunks
				s.chunks = make([]*uintptr, len(s.chunks)+10)
				copy(s.chunks, old)
			}

			// All chunks on the stack have to return here
			runtime.SetFinalizer(p, s.push)
			s.chunks[top+1] = p
			atomic.AddInt32(s.top, 1)
			return
		}
	}
}

func (s *slab) pop() *uintptr {
	for {
		top := atomic.LoadInt32(s.top)
		if top < 0 {
			b := make([]byte, s.size)
			h := (*reflect.SliceHeader)(unsafe.Pointer(&b))
			p := &h.Data
			runtime.SetFinalizer(p, s.push)
			return p // ### return, newly allocated ###
		}

		if atomic.CompareAndSwapInt32(s.head, top, top-1) {
			p := s.chunks[top]
			s.chunks[top] = nil
			atomic.AddInt32(s.top, -1)
			return p
		}
	}
}*/
