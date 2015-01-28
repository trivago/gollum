package producer

import (
	"github.com/artyom/scribe"
	"gollum/shared"
	"time"
)

const (
	scribeBufferGrowSize = 256
)

type scribeMessageBuffer struct {
	message       []*scribe.LogEntry
	messageIdx    int
	contentLen    int
	maxContentLen int
	lastAppend    time.Time
}

func createScribeMessageBuffer(maxContentLen int) *scribeMessageBuffer {
	return &scribeMessageBuffer{
		message:       make([]*scribe.LogEntry, scribeBufferGrowSize),
		messageIdx:    0,
		contentLen:    0,
		maxContentLen: maxContentLen,
		lastAppend:    time.Now(),
	}
}

func (batch *scribeMessageBuffer) append(msg shared.Message, category string, forward bool) bool {
	messageLength := msg.Length(forward)

	if batch.contentLen+messageLength >= batch.maxContentLen {
		return false
	}

	// Grow scribe message array if necessary
	if batch.messageIdx == len(batch.message) {
		temp := batch.message
		batch.message = make([]*scribe.LogEntry, len(batch.message)+scribeBufferGrowSize)
		copy(batch.message, temp)
	}

	logEntry := scribe.LogEntry{
		Category: category,
		Message:  msg.Format(forward),
	}

	batch.message[batch.messageIdx] = &logEntry
	batch.contentLen += messageLength
	batch.lastAppend = time.Now()
	batch.messageIdx++

	return true
}

func (batch *scribeMessageBuffer) appendAndRelease(msg shared.Message, category string, forward bool) bool {
	result := batch.append(msg, category, forward)
	msg.Data.Release()
	return result
}

func (batch scribeMessageBuffer) get() []*scribe.LogEntry {
	return batch.message[:batch.messageIdx]
}

func (batch *scribeMessageBuffer) flush() {
	batch.contentLen = 0
	batch.messageIdx = 0
	batch.lastAppend = time.Now()
}

func (batch scribeMessageBuffer) reachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

func (batch scribeMessageBuffer) reachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastAppend).Seconds() > float64(timeSec)
}
