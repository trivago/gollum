package shared

import (
	"math"
	"sync/atomic"
	"unsafe"
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

// slabsHeader stores an array of slabs of a given size.
// Slab allocation is handled by an index based freelist.
type slabsHeader struct {
	slabLine     [][]byte
	slabSizeByte uint32
	slabsPerLine uint32
	nextFreeIdx  uint32
	access       SpinLock
}

// BytePool implements a convenient way to work with reusable byte buffers that
// are not affected by the garbage collector. A block of bytes is called slabs.
// Slabs are allocated in multiples of 256 byte and allocations are always
// rounded up to fit. Slabs are returned as a SlabHandle which provides a
// reference counted way of managing these resources.
type BytePool struct {
	headers map[uint32]*slabsHeader
	access  *SpinLock
}

// Get the number of slabs per header line.
// A line is currently limited to 1MB or 10 items of a slab.
// So if a slab is 1MB of size a line will be 10MB each.
func getSlabsPerLine(slabSizeByte uint32) uint32 {
	return uint32(math.Max(float64((1<<20)/slabSizeByte), 10.0))
}

// createSlabsHeader initializes a new header for a given slab size and allocates a
// first line of slabs
func createSlabsHeader(slabSizeByte uint32) *slabsHeader {
	header := slabsHeader{
		slabLine:     make([][]byte, 0),
		slabSizeByte: slabSizeByte,
		slabsPerLine: getSlabsPerLine(slabSizeByte),
		nextFreeIdx:  math.MaxUint32,
		access:       SpinLock{},
	}

	header.slabLine = append(header.slabLine, header.createSlabLine())
	return &header
}

// Create a slab line for a specific header
// baseIdx must be set to the current number of slablines * linesize
// An unused slab stores the index of the next element in the freelist
// this is done by reinterpreting the byte array as an integer
func (header *slabsHeader) createSlabLine() []byte {
	lineSizeByte := header.slabSizeByte * header.slabsPerLine
	slabLine := make([]byte, lineSizeByte)

	slabIdx := 0
	globalIdx := uint32(len(header.slabLine)) * header.slabsPerLine

	for i := uint32(1); i < header.slabsPerLine; i++ {
		*(*uint32)(unsafe.Pointer(&slabLine[slabIdx])) = globalIdx + i
		slabIdx += int(header.slabSizeByte)
	}

	*(*uint32)(unsafe.Pointer(&slabLine[slabIdx])) = header.nextFreeIdx
	header.nextFreeIdx = globalIdx

	return slabLine
}

// acquire Returns a new slab from a given header.
// Chunks are taken from the front of a free list. If that list is full a new
// slabline is allocated and all elements are pushed to the freelist beforehand.
func (header *slabsHeader) acquire() *SlabHandle {
	header.access.Lock()
	defer header.access.Unlock()

	if header.nextFreeIdx == math.MaxUint32 {
		header.slabLine = append(header.slabLine, header.createSlabLine())
	}

	slabIdx := header.nextFreeIdx
	lineIdx := slabIdx / header.slabsPerLine
	slabStart := (slabIdx % header.slabsPerLine) * header.slabSizeByte
	slabEnd := slabStart + header.slabSizeByte

	header.nextFreeIdx = *(*uint32)(unsafe.Pointer(&header.slabLine[lineIdx][slabStart]))

	handle := SlabHandle{
		Buffer:   header.slabLine[lineIdx][slabStart:slabEnd],
		Length:   int(header.slabSizeByte),
		refcount: 1,
		index:    slabIdx,
		parent:   header,
	}

	return &handle
}

// release Pushes a slab back to the freelist.
func (header *slabsHeader) release(slab *SlabHandle) {
	header.access.Lock()
	defer header.access.Unlock()

	*(*uint32)(unsafe.Pointer(&slab.Buffer[0])) = header.nextFreeIdx
	header.nextFreeIdx = slab.index
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

// CreateBytePool creates a new, empty pool of headers.
func CreateBytePool() BytePool {
	return BytePool{
		headers: make(map[uint32]*slabsHeader),
		access:  new(SpinLock),
	}
}

// getHeader retrieves the header for the given size and assures that it is
// allocated if it is not there. This function is threadsafe
func (pool BytePool) getHeader(slabSizeByte uint32) *slabsHeader {
	pool.access.Lock()
	defer pool.access.Unlock()

	header, exists := pool.headers[slabSizeByte]
	if !exists {
		header = createSlabsHeader(slabSizeByte)
		pool.headers[slabSizeByte] = header
	}

	return header
}

// Acquire returns a new handle to a slab. Use the member functions of
// SlabHandle to create copies or return it to the pool
func (pool BytePool) Acquire(sizeByte int) *SlabHandle {
	// Round to multiples of 256 byte
	slabSizeByte := uint32(sizeByte)
	if slabSizeByte&0xFF != 0 {
		slabSizeByte = (slabSizeByte & 0xFFFFFF00) + 0x100
	}

	// Fetch a free slab from the header and set the correct size
	header := pool.getHeader(slabSizeByte)
	slab := header.acquire()
	slab.Length = sizeByte

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
