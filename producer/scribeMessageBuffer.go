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

package producer

import (
	"github.com/artyom/scribe"
	"github.com/trivago/gollum/shared"
	"sync"
	"sync/atomic"
	"time"
)

const (
	scribeBufferGrowSize = 256
)

type scribeMessageQueue struct {
	buffer     []*scribe.LogEntry
	contentLen int
	doneCount  uint32
}

func newMessageQueue() scribeMessageQueue {
	return scribeMessageQueue{
		buffer:     make([]*scribe.LogEntry, scribeBufferGrowSize),
		contentLen: 0,
		doneCount:  0,
	}
}

type scribeMessageBuffer struct {
	queue         [2]scribeMessageQueue
	activeSet     uint32
	maxContentLen int
	lastFlush     time.Time
	format        shared.Formatter
	flushing      *sync.Mutex
}

func createScribeMessageBuffer(maxContentLen int, format shared.Formatter) *scribeMessageBuffer {
	return &scribeMessageBuffer{
		queue:         [2]scribeMessageQueue{newMessageQueue(), newMessageQueue()},
		activeSet:     uint32(0),
		maxContentLen: maxContentLen,
		lastFlush:     time.Now(),
		format:        format,
		flushing:      new(sync.Mutex),
	}
}

func (batch *scribeMessageBuffer) append(msg shared.Message, category string) bool {
	activeSet := atomic.AddUint32(&batch.activeSet, 1)
	activeIdx := activeSet >> 31
	messageIdx := (activeSet & 0x7FFFFFFF) - 1

	batch.format.PrepareMessage(msg)
	messageLength := batch.format.GetLength()
	activeQueue := &batch.queue[activeIdx]

	if activeQueue.contentLen+messageLength >= batch.maxContentLen {
		return false
	}

	// Grow scribe message array if necessary
	if messageIdx == uint32(len(activeQueue.buffer)) {
		temp := activeQueue.buffer
		activeQueue.buffer = make([]*scribe.LogEntry, messageIdx+scribeBufferGrowSize)
		copy(activeQueue.buffer, temp)
	}

	logEntry := activeQueue.buffer[messageIdx]
	if logEntry == nil {
		logEntry = new(scribe.LogEntry)
		activeQueue.buffer[messageIdx] = logEntry
	}

	logEntry.Category = category
	logEntry.Message = batch.format.String()

	activeQueue.contentLen += messageLength
	activeQueue.doneCount++

	return true
}

func (batch *scribeMessageBuffer) Append(msg shared.Message, category string) bool {
	result := batch.append(msg, category)
	return result
}

func (batch *scribeMessageBuffer) touch() {
	batch.lastFlush = time.Now()
}

func (batch *scribeMessageBuffer) flush(scribe *scribe.ScribeClient, onError func(error)) {
	if batch.isEmpty() {
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
		// Spin
	}

	go func() {
		defer batch.flushing.Unlock()

		_, err := scribe.Log(flushQueue.buffer[:writerCount])
		flushQueue.contentLen = 0
		flushQueue.doneCount = 0
		batch.touch()

		if err != nil {
			onError(err)
		}
	}()
}

func (batch *scribeMessageBuffer) waitForFlush() {
	batch.flushing.Lock()
	batch.flushing.Unlock()
}

func (batch scribeMessageBuffer) isEmpty() bool {
	return batch.activeSet&0x7FFFFFFF == 0
}

func (batch scribeMessageBuffer) reachedSizeThreshold(size int) bool {
	activeIdx := batch.activeSet >> 31
	return batch.queue[activeIdx].contentLen >= size
}

func (batch scribeMessageBuffer) reachedTimeThreshold(timeout time.Duration) bool {
	return !batch.isEmpty() &&
		time.Since(batch.lastFlush) > timeout
}
