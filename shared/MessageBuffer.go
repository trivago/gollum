package shared

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type messageQueue struct {
	buffer     []byte
	contentLen int
	doneCount  uint32
}

func NewMessageQueue(size int) messageQueue {
	return messageQueue{
		buffer:     make([]byte, size),
		contentLen: 0,
		doneCount:  uint32(0),
	}
}

// MessageBuffer is a helper class for producers to store messages into a stream
// that is flushed into an io.Writer. You can use the Reached* functions to
// determine when a flush should be called.
type MessageBuffer struct {
	delimiter string
	queue     [2]messageQueue
	flushing  *sync.Mutex
	lastFlush time.Time
	activeSet uint32
	format    Formatter
}

// NewMessageBuffer creates a new messagebuffer with a given size (in bytes)
// and a given set of FormatterFlag.
func NewMessageBuffer(size int, format Formatter) *MessageBuffer {
	return &MessageBuffer{
		queue:     [2]messageQueue{NewMessageQueue(size), NewMessageQueue(size)},
		flushing:  new(sync.Mutex),
		lastFlush: time.Now(),
		activeSet: uint32(0),
		format:    format,
	}
}

// Append formats a message and adds it to the buffer.
// If the message does not fit into the buffer this function returns false.
func (batch *MessageBuffer) Append(msg Message) bool {
	activeSet := atomic.AddUint32(&batch.activeSet, 1)
	activeIdx := activeSet >> 31

	messageLength := batch.format.GetLength(msg)
	activeQueue := &batch.queue[activeIdx]

	if activeQueue.contentLen+messageLength >= len(activeQueue.buffer) {
		return false
	}

	batch.format.CopyTo(msg, activeQueue.buffer[activeQueue.contentLen:])
	activeQueue.contentLen += messageLength
	activeQueue.doneCount++

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
// internal state, i.e. the buffer is empty after a call to Flush.
// Writing will be done in a separate go routine to be non-blocking.
func (batch *MessageBuffer) Flush(resource io.Writer, onSuccess func() bool, onError func(error)) {
	if batch.IsEmpty() {
		return // ### return, nothing to do ###
	}

	// Only one flush at a time

	batch.flushing.Lock()

	// Switch the buffers so writers can go on writing

	var flushSet uint32
	if batch.activeSet&0x80000000 != 0 {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0)
	} else {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0x80000000)
	}

	flushIdx := flushSet >> 31
	writerCount := flushSet & 0x7FFFFFFF
	flushQueue := &batch.queue[flushIdx]

	// Wait for remaining writers to finish

	for writerCount != flushQueue.doneCount {
		runtime.Gosched()
	}

	// Write data and reset buffer

	go func() {
		defer batch.flushing.Unlock()
		_, err := resource.Write(flushQueue.buffer[:flushQueue.contentLen])

		if err == nil && onSuccess() {
			flushQueue.contentLen = 0
			flushQueue.doneCount = 0
			batch.Touch()
		} else {
			onError(err)
			// This buffer will be retried during the next flush
		}
	}()
}

// WaitForFlush blocks until the current flush command returns
func (batch *MessageBuffer) WaitForFlush() {
	batch.flushing.Lock()
	batch.flushing.Unlock()
}

// IsEmpty returns true if no data is stored in the buffer
func (batch MessageBuffer) IsEmpty() bool {
	return batch.activeSet&0x7FFFFFFF == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch MessageBuffer) ReachedSizeThreshold(size int) bool {
	activeIdx := batch.activeSet >> 31
	return batch.queue[activeIdx].contentLen >= size
}

// ReachedTimeThreshold returns true if the last flush was more than timeSec ago.
// If there is no data this function returns false.
func (batch MessageBuffer) ReachedTimeThreshold(timeSec int) bool {
	return !batch.IsEmpty() &&
		time.Since(batch.lastFlush).Seconds() > float64(timeSec)
}
