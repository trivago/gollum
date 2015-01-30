package producer

import (
	"github.com/artyom/scribe"
	"github.com/trivago/gollum/shared"
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
	lastFlush     time.Time
	flags         shared.MessageFormatFlag
}

func createScribeMessageBuffer(maxContentLen int, flags shared.MessageFormatFlag) *scribeMessageBuffer {
	return &scribeMessageBuffer{
		message:       make([]*scribe.LogEntry, scribeBufferGrowSize),
		messageIdx:    0,
		contentLen:    0,
		maxContentLen: maxContentLen,
		lastFlush:     time.Now(),
		flags:         flags,
	}
}

func (batch *scribeMessageBuffer) append(msg shared.Message, category string) bool {
	messageLength := msg.Length(batch.flags)

	if batch.contentLen+messageLength >= batch.maxContentLen {
		return false
	}

	// Grow scribe message array if necessary
	if batch.messageIdx == len(batch.message) {
		temp := batch.message
		batch.message = make([]*scribe.LogEntry, len(batch.message)+scribeBufferGrowSize)
		copy(batch.message, temp)
	}

	logEntry := batch.message[batch.messageIdx]

	if logEntry == nil {
		logEntry = new(scribe.LogEntry)
		batch.message[batch.messageIdx] = logEntry
	}

	logEntry.Category = category
	logEntry.Message = msg.Format(batch.flags)

	batch.contentLen += messageLength
	batch.messageIdx++

	return true
}

func (batch *scribeMessageBuffer) appendAndRelease(msg shared.Message, category string) bool {
	result := batch.append(msg, category)
	return result
}

func (batch scribeMessageBuffer) get() []*scribe.LogEntry {
	return batch.message[:batch.messageIdx]
}

func (batch *scribeMessageBuffer) flush() {
	batch.contentLen = 0
	batch.messageIdx = 0
	batch.lastFlush = time.Now()
}

func (batch scribeMessageBuffer) reachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

func (batch scribeMessageBuffer) reachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastFlush).Seconds() > float64(timeSec)
}
