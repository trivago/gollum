package shared

import (
	"math"
	"sync"
	"sync/atomic"
	"unsafe"
)

// SlabHandle stores a reference to a slab (byte buffer of a given size).
// The handle contains a reference counter that will give the slab back to its
// pool when reaching 0. The Acquire and Release member functions implement
// this functionality.
type SlabHandle struct {
	Buffer   []byte
	Length   int
	refcount int32
	index    uint32
	parent   *bytePoolChunk
}

// bytePoolChunk stores an array of slabs of a given size.
// Slab allocation is handled by an index based freelist.
type bytePoolChunk struct {
	slabs       [][]byte
	slabSize    uint32
	slabCount   uint32
	nextFreeIdx uint32
	access      sync.Mutex
}

// BytePool implements a convenient way to work with reusable byte buffers that
// are not affected by the garbage collector. A block of bytes is called slabs.
// Slabs are allocated in multiples of 256 byte and allocations are always
// rounded up to fit. Slabs are returned as a SlabHandle which provides a
// reference counted way of managing these resources.
type BytePool struct {
	chunks map[uint32]*bytePoolChunk
	access *sync.Mutex
}

// Get the number of slabs per chunk line.
// A line is currently limited to 1MB or 10 items of a slab.
// So if a slab is 1MB of size a line will be 10MB each.
func getSlabCount(slabSize uint32) uint32 {
	return uint32(math.Max(float64((1<<20)/slabSize), 10.0))
}

// createChunk initializes a new chunk for a given slab size and allocates a
// first line of slabs
func createChunk(slabSize uint32) *bytePoolChunk {
	chunk := bytePoolChunk{
		slabs:       make([][]byte, 0),
		slabSize:    slabSize,
		slabCount:   getSlabCount(slabSize),
		nextFreeIdx: math.MaxUint32,
		access:      sync.Mutex{},
	}

	chunk.slabs = append(chunk.slabs, chunk.createSlabLine())
	return &chunk
}

// Create a slab line for a specific chunk
// baseIdx must be set to the current number of slablines * linesize
// An unused slab stores the index of the next element in the freelist
// this is done by reinterpreting the byte array as an integer
func (chunk *bytePoolChunk) createSlabLine() []byte {
	lineSize := chunk.slabSize * chunk.slabCount
	slabIdx := 0
	baseIdx := uint32(len(chunk.slabs)) * chunk.slabCount
	slab := make([]byte, lineSize)

	for i := uint32(1); i < chunk.slabCount; i++ {
		*(*uint32)(unsafe.Pointer(&slab[slabIdx])) = baseIdx + i
		slabIdx += int(chunk.slabSize)
	}

	*(*uint32)(unsafe.Pointer(&slab[slabIdx])) = chunk.nextFreeIdx
	chunk.nextFreeIdx = baseIdx
	return slab
}

// acquire Returns a new slab from a given chunk.
// Chunks are taken from the front of a free list. If that list is full a new
// slabline is allocated and all elements are pushed to the freelist beforehand.
func (chunk *bytePoolChunk) acquire() *SlabHandle {
	chunk.access.Lock()
	defer chunk.access.Unlock()

	if chunk.nextFreeIdx == math.MaxUint32 {
		chunk.slabs = append(chunk.slabs, chunk.createSlabLine())
	}

	slabIdx := chunk.nextFreeIdx
	lineIdx := slabIdx / chunk.slabCount
	slabStart := (slabIdx % chunk.slabCount) * chunk.slabSize
	slabEnd := slabStart + chunk.slabSize

	//fmt.Printf("Getting slab %d [%d:%d]\n", slabIdx, slabStart, slabEnd)

	chunk.nextFreeIdx = *(*uint32)(unsafe.Pointer(&chunk.slabs[lineIdx][slabStart]))

	//fmt.Printf("Next slab is %d\n", chunk.nextFreeIdx)

	handle := SlabHandle{
		Buffer:   chunk.slabs[lineIdx][slabStart:slabEnd],
		Length:   int(chunk.slabSize),
		refcount: 1,
		index:    slabIdx,
		parent:   chunk,
	}

	return &handle
}

// release Pushes a slab back to the freelist.
func (chunk *bytePoolChunk) release(slab *SlabHandle) {
	*(*uint32)(unsafe.Pointer(&slab.Buffer[0])) = chunk.nextFreeIdx
	chunk.nextFreeIdx = slab.index
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

// CreateBytePool creates a new, empty pool of chunks.
func CreateBytePool() BytePool {
	return BytePool{
		chunks: make(map[uint32]*bytePoolChunk),
		access: new(sync.Mutex),
	}
}

// Acquire returns a new handle to a slab. Use the member functions of
// SlabHandle to create copies or return it to the pool
func (pool BytePool) Acquire(size int) *SlabHandle {
	// Round to multiples of 256 byte
	slabSize := uint32(size)
	if slabSize&0xFF != 0 {
		slabSize = (slabSize & 0xFFFFFF00) + 0x100
	}

	// Fetch the chunk to modify and be threadsafe
	pool.access.Lock()

	chunk, exists := pool.chunks[slabSize]
	if !exists {
		chunk = createChunk(slabSize)
		pool.chunks[slabSize] = chunk
	}

	pool.access.Unlock()

	// Fetch a free slab from the chunk and set the correct size
	slab := chunk.acquire()
	slab.Length = size
	return slab
}

// AcquireString is a shortcut to acquire a new slab and copy the contents of
// the given string to that buffer
func (pool BytePool) AcquireString(text string) *SlabHandle {
	return pool.AcquireBytes([]byte(text))
}

// AcquireBytes is a shortcut to acquire a new slab and copy the contents of
// the given byte slice to that buffer
func (pool BytePool) AcquireBytes(data []byte) *SlabHandle {
	handle := pool.Acquire(len(data))
	copy(handle.Buffer, data)
	return handle
}
