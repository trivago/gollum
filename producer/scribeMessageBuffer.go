package producer

import (
	"github.com/artyom/scribe"
	"github.com/trivago/gollum/shared"
	"sync"
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
	access        *sync.Mutex
}

func createScribeMessageBuffer(maxContentLen int, flags shared.MessageFormatFlag) *scribeMessageBuffer {
	return &scribeMessageBuffer{
		message:       make([]*scribe.LogEntry, scribeBufferGrowSize),
		messageIdx:    0,
		contentLen:    0,
		maxContentLen: maxContentLen,
		lastFlush:     time.Now(),
		flags:         flags,
		access:        new(sync.Mutex),
	}
}

func (batch *scribeMessageBuffer) append(msg shared.Message, category string) bool {
	batch.access.Lock()
	defer batch.access.Unlock()

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

func (batch *scribeMessageBuffer) touch() {
	batch.lastFlush = time.Now()
}

func (batch *scribeMessageBuffer) flush(scribe *scribe.ScribeClient) error {
	batch.access.Lock()
	defer batch.access.Unlock()

	if !batch.isEmpty() {
		_, err := scribe.Log(batch.message[:batch.messageIdx])
		if err != nil {
			return err
		}
		batch.contentLen = 0
		batch.messageIdx = 0
	}
	batch.touch()
	return nil
}

func (batch scribeMessageBuffer) isEmpty() bool {
	return batch.messageIdx == 0
}

func (batch scribeMessageBuffer) reachedSizeThreshold(size int) bool {
	return batch.contentLen >= size
}

func (batch scribeMessageBuffer) reachedTimeThreshold(timeSec int) bool {
	return batch.contentLen > 0 &&
		time.Since(batch.lastFlush).Seconds() > float64(timeSec)
}
