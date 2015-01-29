package shared

import (
	"io"
	"time"
)

type MessageBuffer struct {
	buffer     []byte
	contentLen int
	lastFlush  time.Time
	flags      MessageFormatFlag
}

func CreateMessageBuffer(size int, flags MessageFormatFlag) *MessageBuffer {
	return &MessageBuffer{
		buffer:     make([]byte, size),
		contentLen: 0,
		lastFlush:  time.Now(),
		flags:      flags,
	}
}

func (batch *MessageBuffer) Append(msg Message) bool {
	messageLength := msg.Length(batch.flags)

	if batch.contentLen+messageLength >= len(batch.buffer) {
		return false
	}

	msg.CopyFormatted(batch.buffer, batch.flags)
	batch.contentLen += messageLength

	return true
}

func (batch *MessageBuffer) AppendAndRelease(msg Message) bool {
	result := batch.Append(msg)
	msg.Data.Release()
	return result
}

func (batch *MessageBuffer) Flush(resource io.Writer) error {
	_, err := resource.Write(batch.buffer[:batch.contentLen])
	if err != nil {
		return err
	}
	batch.contentLen = 0
	batch.lastFlush = time.Now()
	return nil
}

func (batch MessageBuffer) ReachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

func (batch MessageBuffer) ReachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastFlush).Seconds() > float64(timeSec)
}
