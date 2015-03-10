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
	"io"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Internel helper type for frontbuffer/backbuffer storage
type messageQueue struct {
	buffer     []byte
	contentLen int
	doneCount  uint32
}

// ByteStream is an alias to []byte and implements io.Reader as well as
// io.Writer interfaces.
type ByteStream []byte

// StreamBuffer is a helper class for producers to format and store messages
// into a single string that is flushed to an io.Writer.
// You can use the Reached* functions to determine when a flush should be
// called after either reaching a timeout or size threshold.
type StreamBuffer struct {
	delimiter string
	queue     [2]messageQueue
	flushing  *sync.Mutex
	lastFlush time.Time
	activeSet uint32
	format    Formatter
}

func newMessageQueue(size int) messageQueue {
	return messageQueue{
		buffer:     make([]byte, size),
		contentLen: 0,
		doneCount:  uint32(0),
	}
}

// Write implements the io.Writer interface for ByteStream
func (stream *ByteStream) Write(source []byte) (int, error) {
	return copy(*stream, source), nil
}

// Read implements the io.Reader interface for ByteStream
func (stream *ByteStream) Read(target []byte) (int, error) {
	return copy(target, *stream), nil
}

// NewStreamBuffer creates a new StreamBuffer with a given size (in bytes)
// and a given formatter.
func NewStreamBuffer(size int, format Formatter) *StreamBuffer {
	return &StreamBuffer{
		queue:     [2]messageQueue{newMessageQueue(size), newMessageQueue(size)},
		flushing:  new(sync.Mutex),
		lastFlush: time.Now(),
		activeSet: uint32(0),
		format:    format,
	}
}

// Append formats a message and appends it to the internal buffer.
// If the message does not fit into the buffer this function returns false.
// If the message can never fit into the buffer (too large), true is returned
// and an error is logged.
func (batch *StreamBuffer) Append(msg Message) bool {
	activeSet := atomic.AddUint32(&batch.activeSet, 1)
	activeIdx := activeSet >> 31
	activeQueue := &batch.queue[activeIdx]

	// We mark the message as written even if the write fails so that flush
	// does not block after a failed message.
	defer func() { activeQueue.doneCount++ }()

	batch.format.PrepareMessage(msg)
	messageLength := batch.format.GetLength()

	if activeQueue.contentLen+messageLength >= len(activeQueue.buffer) {
		if messageLength > len(activeQueue.buffer) {
			log.Printf("StreamBuffer: Message is too large (%d bytes).", messageLength)
			return true // ### return, cannot be written ever ###
		}
		return false // ### return, cannot be written ###
	}

	batch.format.CopyTo(activeQueue.buffer[activeQueue.contentLen:])
	activeQueue.contentLen += messageLength

	return true
}

// Touch resets the timer queried by ReachedTimeThreshold, i.e. this resets the
// automatic flush timeout
func (batch *StreamBuffer) Touch() {
	batch.lastFlush = time.Now()
}

// Flush writes the content of the buffer to a given resource and resets the
// internal state, i.e. the buffer is empty after a call to Flush.
// Writing will be done in a separate go routine to be non-blocking.
// The validate callback will be called after messages have been written to
// io.Writer. This method should return false if an external check shows that
// no are not all messages arrived at the target.
// The onError callback will be called if the writer returned an error or if
// not all data was written by the writer (the returned length did not match).
// Both callbacks can be nil.
func (batch *StreamBuffer) Flush(resource io.Writer, validate func() bool, onError func(error)) {
	if batch.IsEmpty() {
		return // ### return, nothing to do ###
	}

	// Only one flush at a time
	batch.flushing.Lock()

	// Switch the buffers so writers can go on writing
	// If a previous flush failed we need to continue where we stopped

	var flushSet uint32
	if batch.activeSet&0x80000000 != 0 {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0|batch.queue[0].doneCount)
	} else {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0x80000000|batch.queue[1].doneCount)
	}

	flushIdx := flushSet >> 31
	writerCount := flushSet & 0x7FFFFFFF
	flushQueue := &batch.queue[flushIdx]

	// Wait for remaining writers to finish
	for writerCount != flushQueue.doneCount {
		runtime.Gosched()
	}

	// Write data and reset buffer asynchronously
	go func() {
		defer RecoverShutdown()
		defer batch.flushing.Unlock()

		length, err := resource.Write(flushQueue.buffer[:flushQueue.contentLen])

		if err == nil && length == flushQueue.contentLen && (validate == nil || validate()) {
			flushQueue.contentLen = 0
			flushQueue.doneCount = 0
			batch.Touch()
		} else if onError != nil {
			onError(err)
			// This buffer will be retried during the next flush
		}
	}()
}

// WaitForFlush blocks until the current flush command returns
func (batch *StreamBuffer) WaitForFlush() {
	batch.flushing.Lock()
	batch.flushing.Unlock()
}

// IsEmpty returns true if no data is stored in the buffer
func (batch StreamBuffer) IsEmpty() bool {
	return batch.activeSet&0x7FFFFFFF == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch StreamBuffer) ReachedSizeThreshold(size int) bool {
	activeIdx := batch.activeSet >> 31
	return batch.queue[activeIdx].contentLen >= size
}

// ReachedTimeThreshold returns true if the last flush was more than timeout ago.
// If there is no data this function returns false.
func (batch StreamBuffer) ReachedTimeThreshold(timeout time.Duration) bool {
	return !batch.IsEmpty() &&
		time.Since(batch.lastFlush) > timeout
}
