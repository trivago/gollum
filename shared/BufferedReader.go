package shared

import (
	"bytes"
	"io"
)

// BufferedReader is a helper struct to read from any io.Reader into a byte
// slice. The data can arrive "in pieces" and will be assembled.
type BufferedReader struct {
	data     []byte
	write    func([]byte)
	growSize int
	offset   int
	end      int
	start    int
}

// CreateBufferedReader creates a new buffered reader with a given initial size
// and a callback that is called each time data is parsed as complete.
func NewBufferedReader(size int, callback func([]byte)) BufferedReader {
	return BufferedReader{
		data:     make([]byte, size),
		write:    callback,
		growSize: size,
		offset:   0,
		end:      0,
		start:    0,
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
			msgLength := 0
			for buffer.data[buffer.offset] >= '0' && buffer.data[buffer.offset] <= '9' {
				msgLength = msgLength*10 + int(buffer.data[buffer.offset]-'0')
				buffer.offset++
			}

			if buffer.data[buffer.offset] == ':' {
				buffer.offset++
			} else {
				// Search for next number before exiting, so the next call can
				// pick up at the (hopefully) next message.
				for buffer.data[buffer.offset] < '0' || buffer.data[buffer.offset] > '9' {
					buffer.offset++
				}

				break // ### break, malformed message ###
			}

			// Set the correct offsets for value parsing
			buffer.start = buffer.offset
			buffer.end = buffer.offset + msgLength
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

		buffer.write(buffer.data[buffer.start:buffer.end])
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
		buffer.write(buffer.data[msgStartIdx:msgEndIdx])

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
