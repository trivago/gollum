package shared

import (
	"io"
	"time"
)

// MessageBuffer is a helper class for producers to store messages into a stream
// that is flushed into an io.Writer. You can use the Reached* functions to
// determine when a flush should be called.
type MessageBuffer struct {
	buffer     []byte
	contentLen int
	lastFlush  time.Time
	flags      MessageFormatFlag
	access     *SpinLock
}

// CreateMessageBuffer creates a new messagebuffer with a given size (in bytes)
// and a given set of MessageFormatFlag.
func CreateMessageBuffer(size int, flags MessageFormatFlag) *MessageBuffer {
	return &MessageBuffer{
		buffer:     make([]byte, size),
		contentLen: 0,
		lastFlush:  time.Now(),
		flags:      flags,
		access:     new(SpinLock),
	}
}

// Append formats a message and adds it to the buffer.
// If the message does not fit into the buffer this function returns false.
func (batch *MessageBuffer) Append(msg Message) bool {
	batch.access.Lock()
	defer batch.access.Unlock()

	messageLength := msg.Length(batch.flags)

	if batch.contentLen+messageLength >= len(batch.buffer) {
		return false
	}

	msg.CopyFormatted(batch.buffer[batch.contentLen:], batch.flags)
	batch.contentLen += messageLength

	return true
}

// AppendAndRelease is basically the same as Append but releases the message
// when done.
func (batch *MessageBuffer) AppendAndRelease(msg Message) bool {
	result := batch.Append(msg)
	return result
}

// Touch resets the timer queried by ReachedTimeThreshold, i.e. this resets the
// automatic flush timeout
func (batch *MessageBuffer) Touch() {
	batch.lastFlush = time.Now()
}

// Flush writes the content of the buffer to a given resource and resets the
// internal state, i.e. the buffer is empty after a call to Flush
func (batch *MessageBuffer) Flush(resource io.Writer) error {
	batch.access.Lock()
	defer batch.access.Unlock()

	if !batch.IsEmpty() {
		_, err := resource.Write(batch.buffer[:batch.contentLen])
		if err != nil {
			return err
		}
		batch.contentLen = 0
	}
	batch.Touch()
	return nil
}

// IsEmpty returns true if no data is stored in the buffer
func (batch MessageBuffer) IsEmpty() bool {
	return batch.contentLen == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch MessageBuffer) ReachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

// ReachedTimeThreshold returns true if the last flush was more than timeSec ago.
// If there is no data this function returns false.
func (batch MessageBuffer) ReachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastFlush).Seconds() > float64(timeSec)
}
