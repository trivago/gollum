package shared

import (
	"bytes"
	"io"
)

// BufferedReader is a helper struct to read from any io.Reader into a byte
// slice. The data can arrive "in pieces" and will be assembled.
type BufferedReader struct {
	data     []byte
	write    func([]byte, uint64)
	growSize int
	offset   int
	end      int
	start    int
	sequence int64
}

// NewBufferedReader creates a new buffered reader with a given initial size
// and a callback that is called each time data is parsed as complete.
func NewBufferedReader(size int, callback func([]byte, uint64)) BufferedReader {
	return BufferedReader{
		data:     make([]byte, size),
		write:    callback,
		growSize: size,
		offset:   0,
		end:      0,
		start:    0,
		sequence: 0,
	}
}

// NewBufferedReaderSequence creates a new buffered reader that expects the
// sequence number to be encoded into the message, i.e. is prepended as
// "sequence:".
func NewBufferedReaderSequence(size int, callback func([]byte, uint64)) BufferedReader {
	return BufferedReader{
		data:     make([]byte, size),
		write:    callback,
		growSize: size,
		offset:   0,
		end:      0,
		start:    0,
		sequence: int64(-1),
	}
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
// of the matched string. If the string was not matched properly a size of -1
// and the number 0 is returned.
func readNumberPrefix(data []byte) (uint64, int) {
	prefix := uint64(0)
	index := 0

	for data[index] >= '0' && data[index] <= '9' {
		prefix = prefix*10 + uint64(data[index]-'0')
		index++
	}

	if data[index] == ':' {
		return prefix, index + 1
	}

	return 0, -1
}

// Write a slice of the internal buffer as message to the callback
func (buffer *BufferedReader) post(start int, end int) {
	message := buffer.data[start:end]
	if buffer.sequence < 0 {
		sequence, length := readNumberPrefix(message)
		buffer.write(message[length:], sequence)
	} else {
		buffer.write(message, uint64(buffer.sequence))
		buffer.sequence++
	}
}

// ReadRLE reads from the given reader, expecting runlength encoding, i.e. each
// data block has to be prepended with "length:".
// After the parsing the given number of bytes the data is passed to the callback
// passed to the BufferedReader. If the parsed data does not fit into the
// allocated, internal buffer the buffer is resized.
func (buffer *BufferedReader) ReadRLE(reader io.Reader) error {
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

// Read reads from the given io.Reader until delimiter is reached.
// After the parsing of the delimiter string the data is passed to the callback
// passed to the BufferedReader. If the parsed data does not fit into the
// allocated, internal buffer the buffer is resized.
func (buffer *BufferedReader) Read(reader io.Reader, delimiter string) error {
	bytesRead, err := reader.Read(buffer.data[buffer.offset:])

	if err != nil && bytesRead == 0 {
		return err
	}

	if bytesRead == 0 {
		return nil
	}

	delimiterLen := len(delimiter)
	delimiterBytes := []byte(delimiter)

	// Go through the stream and look for delimiters
	// Execute callback once per delimiter

	parseEndIdx := buffer.offset + bytesRead
	parseStartIdx := buffer.offset
	msgStartIdx := 0

	for parseEndIdx-parseStartIdx > delimiterLen {
		msgEndIdx := bytes.Index(buffer.data[parseStartIdx:parseEndIdx], delimiterBytes)
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
