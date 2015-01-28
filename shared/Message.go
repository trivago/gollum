package shared

import (
	"fmt"
	"hash/fnv"
	"time"
)

type MessageStreamID uint64

var LogInternalStreamID = GetStreamID("_GOLLUM_")
var WildcardStreamID = GetStreamID("*")

const (
	TimestampFormat = "2006-01-02 15:04:05 MST"
)

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	return MessageStreamID(hash.Sum64())
}

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data      *SlabHandle
	StreamID  MessageStreamID
	Timestamp time.Time
}

// CreateMessage creates a new message from a given byte slice
func CreateMessage(pool *BytePool, data []byte, streamID MessageStreamID) Message {
	return Message{
		Data:      pool.AcquireBytes(data),
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
}

// CreateMessage creates a new message from a given string
func CreateMessageFromString(pool *BytePool, text string, streamID MessageStreamID) Message {
	return Message{
		Data:      pool.AcquireString(text),
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
}

// Length calculates the length of the message returned by Format or FormatToCopy
func (msg Message) Length(forward bool) int {
	if forward {
		return msg.Data.Length
	} else {
		return len(TimestampFormat) + 3 + msg.Data.Length
	}
}

// Format converts a Message back to a standardized string format.
func (msg Message) Format(forward bool) string {
	if forward {
		return string(msg.Data.Buffer[:msg.Data.Length])
	}

	return fmt.Sprintf("%s | %s",
		msg.Timestamp.Format(TimestampFormat),
		string(msg.Data.Buffer[:msg.Data.Length]))
}

// CopyFormatted does the same thing as Format but instead of creating a new string
// it copies the result to the given byte slice
func (msg Message) CopyFormatted(buffer []byte, forward bool) {
	if forward {
		copy(buffer, msg.Data.Buffer[:msg.Data.Length])
	} else {
		timestamp := fmt.Sprintf("%s | ", msg.Timestamp.Format(TimestampFormat))
		copy(buffer, []byte(timestamp))
		copy(buffer[len(timestamp):], msg.Data.Buffer[:msg.Data.Length])
	}
}
