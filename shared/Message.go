package shared

import (
	"fmt"
	"hash/fnv"
	"time"
)

// MessageStreamID is the "compiled name" of a stream
type MessageStreamID uint64

// MessageFormatFlag is an enum that is used for formatting messages
type MessageFormatFlag int

// LogInternalStreamID is the ID of the "_GOLLUM_" stream
var LogInternalStreamID = GetStreamID("_GOLLUM_")

// WildcardStreamID is the ID of the "*" stream
var WildcardStreamID = GetStreamID("*")

const (
	// TimestampFormat is the timestamp format string used for messages
	TimestampFormat = "2006-01-02 15:04:05 MST"
	// MessageFormatDefault formats the messages with timestamp and without newline
	MessageFormatDefault = MessageFormatFlag(0)
	// MessageFormatForward formats the message as-is, i.e. without a timestamp
	MessageFormatForward = MessageFormatFlag(1)
	// MessageFormatNewLine adds a newline to the end of the message
	MessageFormatNewLine = MessageFormatFlag(2)
)

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data         string
	Streams      []MessageStreamID
	PinnedStream MessageStreamID
	Timestamp    time.Time
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	return MessageStreamID(hash.Sum64())
}

// CreateMessage creates a new message from a given string
func CreateMessage(text string, streams []MessageStreamID) Message {
	msg := Message{
		Data:         text,
		Streams:      streams,
		PinnedStream: WildcardStreamID,
		Timestamp:    time.Now(),
	}
	return msg
}

// CreateMessageFromSlice creates a new message from a given byte slice
func CreateMessageFromSlice(data []byte, streams []MessageStreamID) Message {
	return Message{
		Data:         string(data),
		Streams:      streams,
		PinnedStream: WildcardStreamID,
		Timestamp:    time.Now(),
	}
}

// CloneAndPin creates a copy of the message and sets the PinnedStream member
// to the given stream. In addition to that the reference counter is increased.
func (msg Message) CloneAndPin(stream MessageStreamID) Message {
	//msg.Data.Acquire()
	msg.PinnedStream = stream
	return msg
}

// IsInternal returns true if a message is posted only to internal streams
func (msg Message) IsInternal() bool {
	for _, value := range msg.Streams {
		if value != LogInternalStreamID {
			return false
		}
	}

	return true
}

// Length calculates the length of the message returned by Format or FormatToCopy
func (msg Message) Length(flags MessageFormatFlag) int {
	length := 0

	if (flags & MessageFormatNewLine) != 0 {
		length = 1
	}

	if (flags & MessageFormatForward) == 0 {
		length += len(TimestampFormat) + 3 + len(msg.Data)
	} else {
		length += len(msg.Data)
	}

	return length
}

// Format converts a Message back to a standardized string format.
func (msg Message) Format(flags MessageFormatFlag) string {
	switch flags {
	default:
		return fmt.Sprintf("%s | %s", msg.Timestamp.Format(TimestampFormat), msg.Data)

	case MessageFormatNewLine:
		return fmt.Sprintf("%s | %s\n", msg.Timestamp.Format(TimestampFormat), msg.Data)

	case MessageFormatForward:
		return msg.Data

	case MessageFormatForward | MessageFormatNewLine:
		return fmt.Sprintf("%s\n", msg.Data)
	}
}

// CopyFormatted does the same thing as Format but instead of creating a new string
// it copies the result to the given byte slice
func (msg Message) CopyFormatted(buffer []byte, flags MessageFormatFlag) {
	formattedString := msg.Format(flags)
	copy(buffer, formattedString)
}
