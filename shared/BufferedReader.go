package shared

import (
	"bytes"
	"io"
)

type BufferedReader struct {
	data     []byte
	write    func([]byte)
	growSize int
	offset   int
}

func CreateBufferedReader(size int, callback func([]byte)) BufferedReader {
	return BufferedReader{
		data:     make([]byte, size),
		write:    callback,
		growSize: size,
		offset:   0,
	}
}

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

		// msgEndIdx is relativ to the slice we passed

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
