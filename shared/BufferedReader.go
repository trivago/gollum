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

import (
	"bytes"
	"io"
)

// BufferedReaderFlags is an enum to configure a buffered reader
type BufferedReaderFlags byte

const (
	// BufferedReaderFlagRLE enables runlength encoded message parsing
	BufferedReaderFlagRLE = 1 << iota
	// BufferedReaderFlagSequence enables sequence encoded message parsing
	BufferedReaderFlagSequence = 1 << iota
)

// BufferedReader is a helper struct to read from any io.Reader into a byte
// slice. The data can arrive "in pieces" and will be assembled.
type BufferedReader struct {
	data      []byte
	write     func([]byte, uint64)
	Read      func(io.Reader) error
	delimiter []byte
	sequence  int64
	growSize  int
	offset    int
	end       int
	start     int
}

// NewBufferedReader creates a new buffered reader with a given initial size
// and a callback that is called each time data is parsed as complete.
// The flags and delimiter passed to NewBufferedReader define how the message
// is parsed from the reader passed to BufferedReader.Read.
func NewBufferedReader(size int, flags BufferedReaderFlags, delimiter string, callback func(msg []byte, sequence uint64)) BufferedReader {
	buffer := BufferedReader{
		data:      make([]byte, size),
		write:     callback,
		delimiter: []byte(delimiter),
		sequence:  0,
		growSize:  size,
		offset:    0,
		end:       0,
		start:     0,
	}

	if flags&BufferedReaderFlagSequence != 0 {
		buffer.sequence = -1
	}

	if flags&BufferedReaderFlagRLE != 0 {
		buffer.Read = buffer.readRLE
	} else {
		buffer.Read = buffer.read
	}

	return buffer
}

// Reset clears the buffer by resetting its internal state
func (buffer *BufferedReader) Reset(sequence uint64) {
	if buffer.sequence >= 0 {
		buffer.sequence = int64(sequence)
	}
	buffer.offset = 0
	buffer.end = 0
	buffer.start = 0
}

// Read "number:" from the data stream and return number as well as the length
// of the matched string. If the string was not matched properly the number 0
// and a length of 0 will be returned.
func readNumberPrefix(data []byte) (uint64, int) {
	prefix, index := Btoi(data)
	if data[index] == ':' {
		index++
	}
	return prefix, index
}

// Write a slice of the internal buffer as message to the callback
func (buffer *BufferedReader) post(start int, end int) {
	message := buffer.data[start:end]

	if buffer.sequence >= 0 {
		buffer.write(message, uint64(buffer.sequence))
		buffer.sequence++
	} else {
		sequence, length := readNumberPrefix(message)
		buffer.write(message[length:], sequence)
	}
}

// readRLE reads from the given reader, expecting runlength encoding, i.e. each
// data block has to be prepended with "length:".
// After the parsing the given number of bytes the data is passed to the callback
// passed to the BufferedReader. If the parsed data does not fit into the
// allocated, internal buffer the buffer is resized.
func (buffer *BufferedReader) readRLE(reader io.Reader) error {
	bytesRead, err := reader.Read(buffer.data[buffer.offset:])

	if err != nil && bytesRead == 0 {
		return err
	}
	if bytesRead == 0 {
		return nil
	}

	// Read message, messages or part of a message
	readEnd := bytesRead + buffer.offset
	for {
		if buffer.offset == 0 {
			// New buffer, parse length
			msgLength, size := readNumberPrefix(buffer.data)

			if size < 0 {
				// Search for next number before exiting, so the next call can
				// pick up at the (hopefully) next message.
				for buffer.data[buffer.offset] < '0' || buffer.data[buffer.offset] > '9' {
					buffer.offset++
				}

				break // ### break, malformed message ###
			}

			// Set the correct offsets for value parsing
			buffer.offset += size
			buffer.start = buffer.offset
			buffer.end = buffer.offset + int(msgLength)
		}

		// Messages might come in parts or be continued with the next read, so
		// we need to test if we're done.

		if readEnd < buffer.end {
			buffer.offset = readEnd

			// Grow if necessary
			bufferSize := len(buffer.data)
			if buffer.offset == bufferSize {
				temp := buffer.data
				buffer.data = make([]byte, bufferSize+buffer.growSize)
				copy(buffer.data, temp)
			}

			break // ### break, Done processing this buffer ###
		}

		buffer.post(buffer.start, buffer.end)
		buffer.offset = 0

		if readEnd == buffer.end {
			break // ### break, nothing left to read ###
		}

		// Resume loop with next message in buffer
		copy(buffer.data, buffer.data[buffer.end:readEnd])
		readEnd -= buffer.end
	}

	return nil
}

// read reads from the given io.Reader until delimiter is reached.
// After the parsing of the delimiter string the data is passed to the callback
// passed to the BufferedReader. If the parsed data does not fit into the
// allocated, internal buffer the buffer is resized.
func (buffer *BufferedReader) read(reader io.Reader) error {
	bytesRead, err := reader.Read(buffer.data[buffer.offset:])

	if err != nil && bytesRead == 0 {
		return err
	}

	if bytesRead == 0 {
		return nil
	}

	delimiterLen := len(buffer.delimiter)

	// Go through the stream and look for delimiters
	// Execute callback once per delimiter

	parseEndIdx := buffer.offset + bytesRead
	parseStartIdx := buffer.offset
	msgStartIdx := 0

	for parseEndIdx-parseStartIdx > delimiterLen {
		msgEndIdx := bytes.Index(buffer.data[parseStartIdx:parseEndIdx], buffer.delimiter)
		if msgEndIdx == -1 {
			break
		}

		// msgEndIdx is relative to the slice we passed
		msgEndIdx += parseStartIdx
		buffer.post(msgStartIdx, msgEndIdx)

		msgStartIdx = msgEndIdx + delimiterLen
		parseStartIdx = msgStartIdx
	}

	// Manage the buffer remains

	if msgStartIdx == 0 {
		// If we did not move at all continue reading. If we don't have any
		// space left, resize the buffer by its original size.
		bufferSize := len(buffer.data)
		if parseEndIdx == bufferSize {
			temp := buffer.data
			buffer.data = make([]byte, bufferSize+buffer.growSize)
			copy(buffer.data, temp)
		}
		buffer.offset = parseEndIdx

	} else if parseStartIdx != parseEndIdx {
		// If we did move but there are remains left in the buffer move them
		// to the start of the buffer and read again
		copy(buffer.data, buffer.data[parseStartIdx:parseEndIdx])
		buffer.offset = parseEndIdx - parseStartIdx
	} else {
		// Everything was written
		buffer.offset = 0
	}

	return nil
}
