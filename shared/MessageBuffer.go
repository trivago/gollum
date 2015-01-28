package shared

import (
	"io"
	"time"
)

type MessageBuffer struct {
	buffer     []byte
	contentLen int
	lastAppend time.Time
}

func CreateMessageBuffer(size int) *MessageBuffer {
	return &MessageBuffer{
		buffer:     make([]byte, size),
		contentLen: 0,
		lastAppend: time.Now(),
	}
}

func (batch *MessageBuffer) Append(msg Message, forward bool) bool {
	messageLength := msg.Length(forward)

	if batch.contentLen+messageLength >= len(batch.buffer) {
		return false
	}

	msg.CopyFormatted(batch.buffer, forward)
	batch.contentLen += messageLength
	batch.lastAppend = time.Now()

	return true
}

func (batch *MessageBuffer) AppendAndRelease(msg Message, forward bool) bool {
	result := batch.Append(msg, forward)
	msg.Data.Release()
	return result
}

func (batch *MessageBuffer) Flush(resource io.Writer) error {
	_, err := resource.Write(batch.buffer[:batch.contentLen])
	if err != nil {
		return err
	}
	batch.contentLen = 0
	batch.lastAppend = time.Now()
	return nil
}

func (batch MessageBuffer) ReachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

func (batch MessageBuffer) ReachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastAppend).Seconds() > float64(timeSec)
}
