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

// NewByteStreamFrom creates a new byte stream that starts with the given
// byte array.
func NewByteStreamFrom(data []byte) ByteStream {
	return ByteStream{
		data:   data,
		offset: len(data),
	}
}

// SetCapacity assures that capacity bytes are available in the buffer, growing
// the managed byte array if needed. The buffer will grow by at least 64 bytes
// if growing is required.
func (stream *ByteStream) SetCapacity(capacity int) {
	if stream.Cap() < capacity {
		current := stream.data
		stream.data = make([]byte, capacity)
		if stream.offset > 0 {
			copy(stream.data, current[:stream.offset])
		}
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

// String returns a string of the underlying byte array containing all written
// data up to this point.
func (stream ByteStream) String() string {
	return string(stream.data[:stream.offset])
}

// Write implements the io.Writer interface.
// This function assures that the capacity of the underlying byte array is
// enough to store the incoming amount of data. Subsequent writes will allways
// append to the end of the stream until Reset() is called.
func (stream *ByteStream) Write(source []byte) (int, error) {
	sourceLen := len(source)
	if sourceLen == 0 {
		return 0, nil
	}

	stream.SetCapacity(stream.offset + sourceLen)
	copy(stream.data[stream.offset:], source[:sourceLen])
	stream.offset += sourceLen

	return sourceLen, nil
}

// WriteString is a convenience wrapper for Write([]byte(source))
func (stream *ByteStream) WriteString(source string) (int, error) {
	return stream.Write([]byte(source))
}

// WriteByte writes a single byte to the stream. Capacity will be ensured.
func (stream *ByteStream) WriteByte(source byte) error {
	stream.SetCapacity(stream.offset + 1)
	stream.data[stream.offset] = source
	stream.offset++
	return nil
}

// Read implements the io.Reader interface.
// The underlying byte array is always copied as a whole.
func (stream *ByteStream) Read(target []byte) (int, error) {
	return copy(target, stream.data), nil
}
