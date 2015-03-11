package shared

// ByteStream is a more lightweight variant of bytes.Buffer.
// The managed byte array is increased to the exact required size and never
// shrinks. Writing moves an internal offset (for appends) but reading always
// starts at offset 0.
type ByteStream struct {
	data   []byte
	offset int
}

// NewByteStream creates a new byte stream of the desired capacity
func NewByteStream(capacity int) ByteStream {
	return ByteStream{
		data:   make([]byte, capacity),
		offset: 0,
	}
}

// SetCapacity assures that capacity bytes are available in the buffer, growing
// the managed byte array if needed.
func (stream *ByteStream) SetCapacity(capacity int) {
	if len(stream.data) < capacity {
		current := stream.data
		stream.data = make([]byte, capacity)
		copy(stream.data, current)
	}
}

// Reset sets the internal write offset to 0
func (stream *ByteStream) Reset() {
	stream.offset = 0
}

// Len returns the length of the underlying array.
// This is equal to len(stream.Bytes()).
func (stream ByteStream) Len() int {
	return stream.offset
}

// Cap returns the capacity of the underlying array.
// This is equal to cap(stream.Bytes()).
func (stream ByteStream) Cap() int {
	return len(stream.data)
}

// Bytes returns a slice of the underlying byte array containing all written
// data up to this point.
func (stream ByteStream) Bytes() []byte {
	return stream.data[:stream.offset]
}

// Write implements the io.Writer interface.
// This function assures that the capacity of the underlying byte array is
// enough to store the incoming amount of data. Subsequent writes will allways
// append to the end of the stream until Reset() is called.
func (stream *ByteStream) Write(source []byte) (int, error) {
	stream.SetCapacity(stream.offset + len(source))

	bytesWritten := copy(stream.data[stream.offset:], source)
	stream.offset += bytesWritten
	return bytesWritten, nil
}

// Read implements the io.Reader interface.
// The underlying byte array is always copied as a whole.
func (stream *ByteStream) Read(target []byte) (int, error) {
	return copy(target, stream.data), nil
}
