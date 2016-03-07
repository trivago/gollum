// Copyright 2015-2016 trivago GmbH
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

package core

import (
	"github.com/trivago/tgo/tsync"
	"time"
)

// MessageStreamID is the "compiled name" of a stream
type MessageStreamID uint64

// MessageStream is the type used for transferring messages between plugins
type MessageBuffer chan Message

const (
	// InvalidStream is used for invalid stream handling and maps to ""
	InvalidStream = ""
	// LogInternalStream is the name of the internal message channel (logs)
	LogInternalStream = "_GOLLUM_"
	// WildcardStream is the name of the "all streams" channel
	WildcardStream = "*"
	// DroppedStream is the name of the stream used to store dropped messages
	DroppedStream = "_DROPPED_"
	// MessageStateOk is returned if the message could be delivered
	MessageStateOk = MessageState(iota)
	// MessageStateTimeout is returned if a message timed out
	MessageStateTimeout = MessageState(iota)
	// MessageStateDiscard is returned if a message should be discarded
	MessageStateDiscard = MessageState(iota)
)

var (
	// InvalidStreamID denotes an invalid stream for function returing stream IDs
	InvalidStreamID = GetStreamID(InvalidStream)
	// LogInternalStreamID is the ID of the "_GOLLUM_" stream
	LogInternalStreamID = GetStreamID(LogInternalStream)
	// WildcardStreamID is the ID of the "*" stream
	WildcardStreamID = GetStreamID(WildcardStream)
	// DroppedStreamID is the ID of the "_DROPPED_" stream
	DroppedStreamID = GetStreamID(DroppedStream)
)

// NewMessageBuffer creates a new message buffer of the given capacity
func NewMessageBuffer(capacity int) MessageBuffer {
	return make(MessageBuffer, capacity)
}

// Push adds a message to the MessageStream.
// waiting for a timeout instead of just blocking.
// Passing a timeout of -1 will discard the message.
// Passing a timout of 0 will always block.
// Messages that time out will be passed to the dropped queue if a Dropped
// consumer exists.
// The source parameter is used when a message is dropped, i.e. it is passed
// to the Drop function.
func (channel MessageBuffer) Push(msg Message, timeout time.Duration) MessageState {
	if timeout == 0 {
		channel <- msg
		return MessageStateOk // ### return, done ###
	}

	start := time.Time{}
	spin := tsync.Spinner{}
	for {
		select {
		case channel <- msg:
			return MessageStateOk // ### return, done ###

		default:
			switch {
			// Start timeout based retries
			case start.IsZero():
				if timeout < 0 {
					return MessageStateDiscard // ### return, discard and ignore ###
				}
				start = time.Now()
				spin = tsync.NewSpinner(tsync.SpinPriorityHigh)

			// Discard message after timeout
			case time.Since(start) > timeout:
				return MessageStateTimeout // ### return, drop and retry ###

			// Yield and try again
			default:
				spin.Yield()
			}
		}
	}
}

// Pop returns a message from the buffer
func (channel MessageBuffer) Pop() (Message, bool) {
	msg, more := <-channel
	return msg, more
}

// Close stops the buffer from being able to recieve messages
func (channel MessageBuffer) Close() {
	close(channel)
}
