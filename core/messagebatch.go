// Copyright 2015-2018 trivago N.V.
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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tsync"
	"sync/atomic"
	"time"
)

// MessageBatch is a helper class for producers to format and store messages
// into a single buffer that is flushed to an io.Writer.
// You can use the Reached* functions to determine whether a flush should be
// called, i.e. if a timeout or size threshold has been reached.
type MessageBatch struct {
	queue     [2]messageBuffer
	flushing  *tsync.WaitGroup
	lastFlush *int64
	activeSet *uint32
	closed    *int32
}

type messageBuffer struct {
	messages  []*Message
	doneCount *uint32
}

const (
	messageBatchIndexShift = 31
	messageBatchCountMask  = 0x7FFFFFFF
	messageBatchIndexMask  = 0x80000000
)

// AssemblyFunc is the function signature for callbacks passed to the Flush
// method.
type AssemblyFunc func([]*Message)

// NewMessageBatch creates a new MessageBatch with a given size (in bytes)
// and a given formatter.
func NewMessageBatch(maxMessageCount int) MessageBatch {
	now := time.Now().Unix()
	return MessageBatch{
		queue:     [2]messageBuffer{newMessageBuffer(maxMessageCount), newMessageBuffer(maxMessageCount)},
		flushing:  new(tsync.WaitGroup),
		lastFlush: &now,
		activeSet: new(uint32),
		closed:    new(int32),
	}
}

func newMessageBuffer(maxMessageCount int) messageBuffer {
	return messageBuffer{
		messages:  make([]*Message, maxMessageCount),
		doneCount: new(uint32),
	}
}

// Len returns the length of one buffer
func (batch *MessageBatch) Len() int {
	return len(batch.queue[0].messages)
}

// The number of elements in the active buffer
func (batch *MessageBatch) getActiveBufferCount() int {
	return int(atomic.LoadUint32(batch.activeSet) & 0x7FFFFFFF)
}

// Append formats a message and appends it to the internal buffer.
// If the message does not fit into the buffer this function returns false.
// If the message can never fit into the buffer (too large), true is returned
// and an error is logged.
func (batch *MessageBatch) Append(msg *Message) bool {
	if batch.IsClosed() {
		return false // ### return, closed ###
	}

	activeSet := atomic.AddUint32(batch.activeSet, 1)
	activeIdx := activeSet >> messageBatchIndexShift
	activeQueue := &batch.queue[activeIdx]
	ticketIdx := (activeSet & messageBatchCountMask) - 1

	// We mark the message as written even if the write fails so that flush
	// does not block after a failed message.
	defer func() { atomic.AddUint32(activeQueue.doneCount, 1) }()

	if ticketIdx >= uint32(len(activeQueue.messages)) {
		return false // ### return, queue is full ###
	}

	activeQueue.messages[ticketIdx] = msg
	return true
}

// AppendOrBlock works like Append but will block until Append returns true.
// If the batch was closed during this call, false is returned.
func (batch *MessageBatch) AppendOrBlock(msg *Message) bool {
	spin := tsync.NewSpinner(tsync.SpinPriorityMedium)
	for !batch.IsClosed() {
		if batch.Append(msg) {
			return true // ### return, success ###
		}
		spin.Yield()
	}

	return false
}

// AppendOrFlush is a common combinatorial pattern of Append and AppendOrBlock.
// If append fails, flush is called, followed by AppendOrBlock if blocking is
// allowed. If AppendOrBlock fails (or blocking is not allowed) the message is
// sent to the fallback.
func (batch *MessageBatch) AppendOrFlush(msg *Message, flushBuffer func(), canBlock func() bool, tryFallback func(*Message)) {
	if !batch.Append(msg) {
		flushBuffer()
		if !canBlock() || !batch.AppendOrBlock(msg) {
			tryFallback(msg)
		}
	}
}

// Touch resets the timer queried by ReachedTimeThreshold, i.e. this resets the
// automatic flush timeout
func (batch *MessageBatch) Touch() {
	atomic.StoreInt64(batch.lastFlush, time.Now().Unix())
}

// Close disables Append, calls flush and waits for this call to finish.
// Timeout is passed to WaitForFlush.
func (batch *MessageBatch) Close(assemble AssemblyFunc, timeout time.Duration) {
	atomic.StoreInt32(batch.closed, 1)
	batch.Flush(assemble)
	batch.WaitForFlush(timeout)
}

// IsClosed returns true of Close has been called at least once.
func (batch MessageBatch) IsClosed() bool {
	return atomic.LoadInt32(batch.closed) != 0
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
func (batch *MessageBatch) Flush(assemble AssemblyFunc) {
	if batch.IsEmpty() {
		return // ### return, nothing to do ###
	}

	// Only one flush at a time
	batch.flushing.IncWhenDone()

	// Switch the buffers so writers can go on writing
	flushSet := atomic.SwapUint32(batch.activeSet, (atomic.LoadUint32(batch.activeSet)&messageBatchIndexMask)^messageBatchIndexMask)

	flushIdx := flushSet >> messageBatchIndexShift
	writerCount := flushSet & messageBatchCountMask
	flushQueue := &batch.queue[flushIdx]
	spin := tsync.NewSpinner(tsync.SpinPriorityHigh)

	// Wait for remaining writers to finish
	for writerCount != atomic.LoadUint32(flushQueue.doneCount) {
		spin.Yield()
	}

	// Write data and reset buffer asynchronously
	go tgo.WithRecoverShutdown(func() {
		defer batch.flushing.Done()

		messageCount := tmath.MinI(int(writerCount), len(flushQueue.messages))
		assemble(flushQueue.messages[:messageCount])
		atomic.StoreUint32(flushQueue.doneCount, 0)
		batch.Touch()
	})
}

// AfterFlushDo calls a function after a currently running flush is done.
// It also blocks any flush during the execution of callback.
// Returns the error returned by callback
func (batch *MessageBatch) AfterFlushDo(callback func() error) error {
	batch.flushing.IncWhenDone()
	defer batch.flushing.Done()
	return callback()
}

// WaitForFlush blocks until the current flush command returns.
// Passing a timeout > 0 will unblock this function after the given duration at
// the latest.
func (batch *MessageBatch) WaitForFlush(timeout time.Duration) {
	batch.flushing.WaitFor(timeout)
}

// IsEmpty returns true if no data is stored in the front buffer, i.e. if no data
// is scheduled for flushing.
func (batch MessageBatch) IsEmpty() bool {
	return atomic.LoadUint32(batch.activeSet)&messageBatchCountMask == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedSizeThreshold(size int) bool {
	activeIdx := atomic.LoadUint32(batch.activeSet) >> messageBatchIndexShift
	threshold := uint32(tmath.MaxI(size, len(batch.queue[activeIdx].messages)))
	return atomic.LoadUint32(batch.queue[activeIdx].doneCount) >= threshold
}

// ReachedTimeThreshold returns true if the last flush was more than timeout ago.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedTimeThreshold(timeout time.Duration) bool {
	lastFlush := time.Unix(atomic.LoadInt64(batch.lastFlush), 0)
	return !batch.IsEmpty() && time.Since(lastFlush) > timeout
}
