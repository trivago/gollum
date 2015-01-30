package shared

// SlabPool implements a convenient way to work with reusable byte buffers that
// are not affected by the garbage collector. A block of bytes is called slabs.
// Slabs are allocated in multiples of 256 byte and allocations are always
// rounded up to fit. Slabs are returned as a SlabHandle which provides a
// reference counted way of managing these resources.
type SlabPool struct {
	headers map[uint32]*slabsHeader
	access  *SpinLock
}

// CreateSlabPool creates a new, empty pool of headers.
func CreateSlabPool() SlabPool {
	return SlabPool{
		headers: make(map[uint32]*slabsHeader),
		access:  new(SpinLock),
	}
}

// getHeader retrieves the header for the given size and assures that it is
// allocated if it is not there. This function is threadsafe
func (pool SlabPool) getHeader(slabSizeByte uint32) *slabsHeader {
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
func (pool SlabPool) Acquire(sizeByte int) *SlabHandle {
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
func (pool SlabPool) AcquireString(text string) *SlabHandle {
	return pool.AcquireBytes([]byte(text))
}

// AcquireBytes is a shortcut to acquire a new slab and copy the contents of
// the given byte slice to that buffer
func (pool SlabPool) AcquireBytes(data []byte) *SlabHandle {
	handle := pool.Acquire(len(data))
	copy(handle.Buffer, data)
	return handle
}
