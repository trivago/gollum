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

package core

import (
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Internel helper type for frontbuffer/backbuffer storage
type messageQueue struct {
	buffer     []byte
	contentLen int32
	doneCount  uint32
}

// MessageBatch is a helper class for producers to format and store messages
// into a single buffer that is flushed to an io.Writer.
// You can use the Reached* functions to determine whether a flush should be
// called, i.e. if a timeout or size threshold has been reached.
type MessageBatch struct {
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

func (queue *messageQueue) reset() {
	queue.contentLen = 0
	queue.doneCount = 0
}

// NewMessageBatch creates a new MessageBatch with a given size (in bytes)
// and a given formatter.
func NewMessageBatch(size int, format Formatter) *MessageBatch {
	return &MessageBatch{
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
func (batch *MessageBatch) Append(msg Message) bool {
	activeSet := atomic.AddUint32(&batch.activeSet, 1)
	activeIdx := activeSet >> 31
	activeQueue := &batch.queue[activeIdx]

	// We mark the message as written even if the write fails so that flush
	// does not block after a failed message.
	defer func() { activeQueue.doneCount++ }()

	payload, _ := batch.format.Format(msg)
	messageLength := len(payload)
	var currentOffset, nextOffset int

	// There might be multiple threads writing to the queue, so try to get a
	// write window (lockless)
	for {
		currentOffset = int(activeQueue.contentLen)
		nextOffset = currentOffset + messageLength

		if nextOffset > len(activeQueue.buffer) {
			if messageLength > len(activeQueue.buffer) {
				Log.Warning.Printf("MessageBatch: Message is too large (%d bytes).", messageLength)
				return true // ### return, cannot be written ever ###
			}
			return false // ### return, queue is full ###
		}

		if atomic.CompareAndSwapInt32(&activeQueue.contentLen, int32(currentOffset), int32(nextOffset)) {
			break // break, got the window
		}
	}

	copy(activeQueue.buffer[currentOffset:], payload)
	return true
}

// Touch resets the timer queried by ReachedTimeThreshold, i.e. this resets the
// automatic flush timeout
func (batch *MessageBatch) Touch() {
	batch.lastFlush = time.Now()
}

// Flush writes the content of the buffer to a given resource and resets the
// internal state, i.e. the buffer is empty after a call to Flush.
// Writing will be done in a separate go routine to be non-blocking.
//
// The validate callback will be called after messages have been successfully
// written to the io.Writer.
// If validate returns false the buffer will not be resetted (automatic retry).
// If validate is nil a return value of true is assumed (buffer reset).
//
// The onError callback will be called if the io.Writer returned an error.
// If onError returns false the buffer will not be resetted (automatic retry).
// If onError is nil a return value of true is assumed (buffer reset).
func (batch *MessageBatch) Flush(resource io.Writer, validate func() bool, onError func(error) bool) {
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
		defer shared.RecoverShutdown()
		defer batch.flushing.Unlock()

		_, err := resource.Write(flushQueue.buffer[:flushQueue.contentLen])

		if err == nil {
			if validate == nil || validate() {
				flushQueue.reset()
			}
		} else {
			if onError == nil || onError(err) {
				flushQueue.reset()
			}
		}

		batch.Touch()
	}()
}

// WaitForFlush blocks until the current flush command returns.
// Passing a timeout > 0 will unblock this function after the given duration at
// the latest.
func (batch *MessageBatch) WaitForFlush(timeout time.Duration) {
	flushed := int32(0)
	if timeout > 0 {
		time.AfterFunc(timeout, func() {
			if atomic.CompareAndSwapInt32(&flushed, 0, 1) {
				batch.flushing.Unlock()
			}
		})
	}

	batch.flushing.Lock()
	if atomic.CompareAndSwapInt32(&flushed, 0, 1) {
		batch.flushing.Unlock()
	}
}

// IsEmpty returns true if no data is stored in the buffer
func (batch MessageBatch) IsEmpty() bool {
	return batch.activeSet&0x7FFFFFFF == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedSizeThreshold(size int) bool {
	activeIdx := batch.activeSet >> 31
	return batch.queue[activeIdx].contentLen >= int32(size)
}

// ReachedTimeThreshold returns true if the last flush was more than timeout ago.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedTimeThreshold(timeout time.Duration) bool {
	return !batch.IsEmpty() &&
		time.Since(batch.lastFlush) > timeout
}
