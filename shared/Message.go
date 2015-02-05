package shared

import (
	"hash/fnv"
	"time"
)

// MessageStreamID is the "compiled name" of a stream
type MessageStreamID uint64

const (
	// LogInternalStream is the name of the internal message channel (logs)
	LogInternalStream = "_GOLLUM_"
	// WildcardStream is the name of the "all streams" channel
	WildcardStream = "*"
)

// LogInternalStreamID is the ID of the "_GOLLUM_" stream
var LogInternalStreamID = GetStreamID(LogInternalStream)

// WildcardStreamID is the ID of the "*" stream
var WildcardStreamID = GetStreamID(WildcardStream)

const (
	// DefaultTimestamp is the timestamp format string used for messages
	DefaultTimestamp = "2006-01-02 15:04:05 MST"
	// DefaultDelimiter is the default end of message delimiter
	DefaultDelimiter = "\n"
)

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data         string
	Streams      []MessageStreamID
	PinnedStream MessageStreamID
	Timestamp    time.Time
}

// MessageFormat is the interface definition for message formatters
type MessageFormat interface {
	GetLength(msg Message) int
	ToString(msg Message) string
	ToBuffer(msg Message, dest []byte)
}

// MessageProvider is the interface definition for anything that provides a
// channel of messages
type MessageProvider interface {
	Messages() <-chan Message
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
