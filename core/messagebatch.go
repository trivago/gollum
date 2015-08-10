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
	"github.com/trivago/gollum/shared"
	"sync/atomic"
	"time"
)

// MessageBatch is a helper class for producers to format and store messages
// into a single buffer that is flushed to an io.Writer.
// You can use the Reached* functions to determine whether a flush should be
// called, i.e. if a timeout or size threshold has been reached.
type MessageBatch struct {
	queue     [2]messageQueue
	flushing  *shared.WaitGroup
	lastFlush time.Time
	activeSet uint32
	closed    bool
}

type messageQueue struct {
	messages  []Message
	doneCount uint32
}

// AssemblyFunc is the function signature for callbacks passed to the Flush
// method.
type AssemblyFunc func([]Message)

// NewMessageBatch creates a new MessageBatch with a given size (in bytes)
// and a given formatter.
func NewMessageBatch(maxMessageCount int) MessageBatch {
	return MessageBatch{
		queue:     [2]messageQueue{newMessageQueue(maxMessageCount), newMessageQueue(maxMessageCount)},
		flushing:  new(shared.WaitGroup),
		lastFlush: time.Now(),
		activeSet: uint32(0),
		closed:    false,
	}
}

func newMessageQueue(maxMessageCount int) messageQueue {
	return messageQueue{
		messages:  make([]Message, maxMessageCount),
		doneCount: uint32(0),
	}
}

// Append formats a message and appends it to the internal buffer.
// If the message does not fit into the buffer this function returns false.
// If the message can never fit into the buffer (too large), true is returned
// and an error is logged.
func (batch *MessageBatch) Append(msg Message) bool {
	if batch.IsClosed() {
		return false // ### return, closed ###
	}

	activeSet := atomic.AddUint32(&batch.activeSet, 1)
	activeIdx := activeSet >> 31
	activeQueue := &batch.queue[activeIdx]
	ticketIdx := (activeSet & 0x7FFFFFFF) - 1

	// We mark the message as written even if the write fails so that flush
	// does not block after a failed message.
	defer func() { atomic.AddUint32(&activeQueue.doneCount, 1) }()

	if ticketIdx >= uint32(len(activeQueue.messages)) {
		return false // ### return, queue is full ###
	}

	activeQueue.messages[ticketIdx] = msg
	return true
}

// AppendOrBlock works like Append but will block until Append returns true.
// If the batch was closed during this call, false is returned.
func (batch *MessageBatch) AppendOrBlock(msg Message) bool {
	spin := shared.NewSpinner(shared.SpinPriorityMedium)
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
// dropped.
func (batch *MessageBatch) AppendOrFlush(msg Message, flushBuffer func(), canBlock func() bool, drop func(Message)) {
	if !batch.Append(msg) {
		flushBuffer()
		if !canBlock() || !batch.AppendOrBlock(msg) {
			drop(msg)
		}
	}
}

// Touch resets the timer queried by ReachedTimeThreshold, i.e. this resets the
// automatic flush timeout
func (batch *MessageBatch) Touch() {
	batch.lastFlush = time.Now()
}

// Close disables Append, calls flush and waits for this call to finish.
// Timeout is passed to WaitForFlush.
func (batch *MessageBatch) Close(assemble AssemblyFunc, timeout time.Duration) {
	batch.closed = true
	batch.Flush(assemble)
	batch.WaitForFlush(timeout)
}

// IsClosed returns true of Close has been called at least once.
func (batch MessageBatch) IsClosed() bool {
	return batch.closed
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
	var flushSet uint32
	if batch.activeSet&0x80000000 != 0 {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0)
	} else {
		flushSet = atomic.SwapUint32(&batch.activeSet, 0x80000000)
	}

	flushIdx := flushSet >> 31
	writerCount := flushSet & 0x7FFFFFFF
	flushQueue := &batch.queue[flushIdx]
	spin := shared.NewSpinner(shared.SpinPriorityHigh)

	// Wait for remaining writers to finish
	for writerCount != flushQueue.doneCount {
		spin.Yield()
	}

	// Write data and reset buffer asynchronously
	go shared.DontPanic(func() {
		defer batch.flushing.Done()

		messageCount := shared.MinI(int(flushQueue.doneCount), len(flushQueue.messages))
		assemble(flushQueue.messages[:messageCount])
		flushQueue.doneCount = 0
		batch.Touch()
	})
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
	return batch.activeSet&0x7FFFFFFF == 0
}

// ReachedSizeThreshold returns true if the bytes stored in the buffer are
// above or equal to the size given.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedSizeThreshold(size int) bool {
	activeIdx := batch.activeSet >> 31
	threshold := uint32(shared.MaxI(size, len(batch.queue[activeIdx].messages)))
	return batch.queue[activeIdx].doneCount >= threshold
}

// ReachedTimeThreshold returns true if the last flush was more than timeout ago.
// If there is no data this function returns false.
func (batch MessageBatch) ReachedTimeThreshold(timeout time.Duration) bool {
	return !batch.IsEmpty() && time.Since(batch.lastFlush) > timeout
}
