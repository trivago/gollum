package shared

import (
	"sync/atomic"
)

// SlabHandle stores a reference to a slab (byte buffer of a given size).
// The handle contains a reference counter that will give the slab back to its
// pool when reaching 0. The Acquire and Release member functions implement
// this functionality.
type SlabHandle struct {
	Buffer   []byte
	parent   *slabsHeader
	Length   int
	refcount int32
	index    uint32
}

// Acquire increments the internal reference counter and returns the handle
// passed to the function. Use this function before copying a slab.
func (slab *SlabHandle) Acquire() *SlabHandle {
	atomic.AddInt32(&slab.refcount, 1)
	return slab
}

// Release decrements the internal reference counter and sends the handle back
// to the pool if the counter reaches 0. Use this function if you don't need
// the slab handle any more.
func (slab *SlabHandle) Release() {
	if atomic.AddInt32(&slab.refcount, -1) == 0 {
		slab.parent.release(slab)
	}
}
