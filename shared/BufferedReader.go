package shared

import (
	"bytes"
	"io"
)

type BufferedReader struct {
	data     []byte
	callback func([]byte)
	growSize int
	offset   int
}

func CreateBufferedReader(size int, callback func([]byte)) BufferedReader {
	return BufferedReader{
		data:     make([]byte, size),
		callback: callback,
		growSize: size,
		offset:   0,
	}
}

func (buffer *BufferedReader) Read(reader io.Reader, delimiter string) error {
	bytesRead, err := reader.Read(buffer.data[buffer.offset:])
	if err != nil {
		return err
	}

	if bytesRead == 0 {
		return nil
	}

	delimiterLen := len(delimiter)
	delimiterBytes := []byte(delimiter)

	// Go through the stream and look for delimiters
	// Execute callback once per delimiter

	endIdx := buffer.offset + bytesRead
	startIdx := 0
	partEndIdx := bytes.Index(buffer.data[buffer.offset:endIdx], delimiterBytes)

	for partEndIdx != -1 {
		buffer.callback(buffer.data[startIdx:partEndIdx])

		startIdx = partEndIdx + delimiterLen
		partEndIdx = bytes.Index(buffer.data[startIdx:endIdx], delimiterBytes)
	}

	// Manage the buffer remains

	if startIdx == 0 {
		// If we did not move at all continue reading. If we don't have any
		// space left, resize the buffer by its original size.

		bufferSize := len(buffer.data)
		if endIdx == bufferSize {
			temp := buffer.data
			buffer.data = make([]byte, bufferSize+buffer.growSize)
			copy(buffer.data, temp)
		}
		buffer.offset = endIdx

	} else if startIdx != endIdx {
		// If we did move but there are remains left in the buffer move them
		// to the start of the buffer and read again

		copy(buffer.data, buffer.data[startIdx:endIdx])
		buffer.offset = endIdx - startIdx
	} else {
		// Everything was written

		buffer.offset = 0
	}

	return nil
}
