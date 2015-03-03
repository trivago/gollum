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

package shared

import (
	"hash/fnv"
	"runtime"
	"time"
)

// MessageStreamID is the "compiled name" of a stream
type MessageStreamID uint64

const (
	// LogInternalStream is the name of the internal message channel (logs)
	LogInternalStream = "_GOLLUM_"
	// WildcardStream is the name of the "all streams" channel
	WildcardStream = "*"
	// DroppedStream is the name of the stream used to store dropped messages
	DroppedStream = "_DROPPED_"
)

// LogInternalStreamID is the ID of the "_GOLLUM_" stream
var LogInternalStreamID = GetStreamID(LogInternalStream)

// WildcardStreamID is the ID of the "*" stream
var WildcardStreamID = GetStreamID(WildcardStream)

// DroppedStreamID is the ID of the "_DROPPED_" stream
var DroppedStreamID = GetStreamID(DroppedStream)

// dropConsumer contains the list of registered channels that recieve dropped
// messages
var dropConsumer = make([]chan<- Message, 0)

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	Data          []byte
	streams       []MessageStreamID
	CurrentStream MessageStreamID
	Timestamp     time.Time
	Sequence      uint64
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	return MessageStreamID(hash.Sum64())
}

// NewMessage creates a new message from a given string
func NewMessage(text string, streams []MessageStreamID, sequence uint64) Message {
	msg := Message{
		Data:          []byte(text),
		streams:       streams,
		CurrentStream: WildcardStreamID,
		Timestamp:     time.Now(),
		Sequence:      sequence,
	}
	return msg
}

// NewMessageFromSlice creates a new message from a given byte slice
func NewMessageFromSlice(data []byte, streams []MessageStreamID, sequence uint64) Message {
	return Message{
		Data:          data,
		streams:       streams,
		CurrentStream: WildcardStreamID,
		Timestamp:     time.Now(),
		Sequence:      sequence,
	}
}

// SetCurrentStream modifies the CurrentStream member if the stream is NOT
// _DROPPED_. In that case the current stream member contains the stream that
// caused the dropped.
func (msg *Message) SetCurrentStream(stream MessageStreamID) {
	if stream != DroppedStreamID {
		msg.CurrentStream = stream
	}
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
		if value != LogInternalStreamID && value != DroppedStreamID {
			return false
		}
	}
	return true
}

// RegisterDropConsumer registers a consumer channel to the list of listeners
// accepting dropped messages.
func RegisterDropConsumer(cons ConsumerBase) {
	dropConsumer = append(dropConsumer, cons.messages)
}

// Drop marks a message as dropped and pushes it into the dropped messages
// channel. This channel may block.
func Drop(msg Message) {
	if len(dropConsumer) > 0 {
		msg.streams = []MessageStreamID{DroppedStreamID}
		for _, ch := range dropConsumer {
			ch <- msg
		}
	}
}

// PostMessage is a convenience function to push a message to a channel while
// waiting for a timeout instead of just blocking.
// Passing a timeout of -1 which will discard the message.
// Passing a timout of 0 will always block.
func PostMessage(channel chan<- Message, msg Message, timeout time.Duration) {
	if timeout == 0 {
		channel <- msg
	} else {
		var start *time.Time
		for {
			select {
			case channel <- msg:
				return

			default:
				switch {
				// Start timeout based retries
				case start == nil:
					if timeout < 0 {
						return
					}
					now := time.Now()
					start = &now
					fallthrough

				// Yield and try again
				default:
					runtime.Gosched()

				// Discard message after timeout
				case time.Since(*start) > timeout:
					Drop(msg)
					return
				}
			}
		}
	}
}
