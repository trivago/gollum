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

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data          []byte
	streams       []MessageStreamID
	CurrentStream MessageStreamID
	Timestamp     time.Time
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	return MessageStreamID(hash.Sum64())
}

// NewMessage creates a new message from a given string
func NewMessage(text string, streams []MessageStreamID) Message {
	msg := Message{
		Data:          []byte(text),
		streams:       streams,
		CurrentStream: WildcardStreamID,
		Timestamp:     time.Now(),
	}
	return msg
}

// NewMessageFromSlice creates a new message from a given byte slice
func NewMessageFromSlice(data []byte, streams []MessageStreamID) Message {
	return Message{
		Data:          data,
		streams:       streams,
		CurrentStream: WildcardStreamID,
		Timestamp:     time.Now(),
	}
}

// CopyAndPin creates a copy of the message and sets the CurrentStream member
// to the given stream. In addition to that the reference counter is increased.
func (msg Message) CopyAndPin(stream MessageStreamID) Message {
	msg.CurrentStream = stream
	return msg
}

// ForEachStream iterates over all streams of this message and calls the given
// callback. If the callback returns false the iteration will stop immediately.
func (msg *Message) ForEachStream(callback func(stream MessageStreamID) bool) {
	for _, value := range msg.streams {
		if !callback(value) {
			return
		}
	}
}

// IsInternalOnly returns true if a message is posted only to internal streams
func (msg Message) IsInternalOnly() bool {
	for _, value := range msg.streams {
		if value != LogInternalStreamID {
			return false
		}
	}
	return true
}
