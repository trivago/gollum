package shared

import (
	"fmt"
	"hash/fnv"
	"time"
)

type MessageStreamID uint64
type MessageFormatFlag int

var LogInternalStreamID = GetStreamID("_GOLLUM_")
var WildcardStreamID = GetStreamID("*")

const (
	TimestampFormat      = "2006-01-02 15:04:05 MST"
	MessageFormatForward = MessageFormatFlag(1)
	MessageFormatNewLine = MessageFormatFlag(2)
)

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data      *SlabHandle
	StreamID  MessageStreamID
	Timestamp time.Time
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	return MessageStreamID(hash.Sum64())
}

// CreateMessage creates a new message from a given byte slice
func CreateMessage(pool *BytePool, data []byte, streamID MessageStreamID) Message {
	return Message{
		Data:      pool.AcquireBytes(data),
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
}

// CreateMessageFromString creates a new message from a given string
func CreateMessageFromString(pool *BytePool, text string, streamID MessageStreamID) Message {
	return Message{
		Data:      pool.AcquireString(text),
		StreamID:  streamID,
		Timestamp: time.Now(),
	}
}

// Length calculates the length of the message returned by Format or FormatToCopy
func (msg Message) Length(flags MessageFormatFlag) int {
	var length int

	if (flags & MessageFormatNewLine) == 0 {
		length = 0
	} else {
		length = 1
	}

	if (flags & MessageFormatForward) != 0 {
		length += msg.Data.Length
	} else {
		length += len(TimestampFormat) + 3 + msg.Data.Length
	}

	return length
}

// Format converts a Message back to a standardized string format.
func (msg Message) Format(flags MessageFormatFlag) string {
	switch flags {
	default:
		return fmt.Sprintf("%s | %s", msg.Timestamp.Format(TimestampFormat), msg.Data.Buffer[:msg.Data.Length])

	case MessageFormatNewLine:
		return fmt.Sprintf("%s | %s\n", msg.Timestamp.Format(TimestampFormat), msg.Data.Buffer[:msg.Data.Length])

	case MessageFormatForward:
		return string(msg.Data.Buffer[:msg.Data.Length])

	case MessageFormatForward | MessageFormatNewLine:
		return fmt.Sprintf("%s\n", msg.Data.Buffer[:msg.Data.Length])
	}
}

// CopyFormatted does the same thing as Format but instead of creating a new string
// it copies the result to the given byte slice
func (msg Message) CopyFormatted(buffer []byte, flags MessageFormatFlag) {
	switch flags {
	default:
		formattedString := msg.Format(flags)
		copy(buffer, formattedString)

	case MessageFormatForward:
		copy(buffer, msg.Data.Buffer[:msg.Data.Length])

	case MessageFormatForward | MessageFormatNewLine:
		copy(buffer, msg.Data.Buffer[:msg.Data.Length])
		msg.Data.Buffer[msg.Data.Length] = '\n'
	}
}
