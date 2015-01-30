package shared

import (
	"math"
	"unsafe"
)

// slabsHeader stores an array of slabs of a given size.
// Slab allocation is handled by an index based freelist.
type slabsHeader struct {
	slabLine     [][]byte
	slabSizeByte uint32
	slabsPerLine uint32
	nextFreeIdx  uint32
	access       SpinLock
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
