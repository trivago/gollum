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

// MessageQueue is the type used for transferring messages between plugins
type MessageQueue chan *Message

const (
	// InvalidStream is used for invalid stream handling and maps to ""
	InvalidStream = ""
	// LogInternalStream is the name of the internal message channel (logs)
	LogInternalStream = "_GOLLUM_"
	// WildcardStream is the name of the "all routers" channel
	WildcardStream = "*"
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
)

// NewMessageQueue creates a new message buffer of the given capacity
func NewMessageQueue(capacity int) MessageQueue {
	return make(MessageQueue, capacity)
}

// Push adds a message to the MessageStream.
// waiting for a timeout instead of just blocking.
// Passing a timeout of -1 will discard the message.
// Passing a timout of 0 will always block.
// Messages that time out will be passed to the sent to the fallback queue if a Dropped
// consumer exists.
// The source parameter is used when a message is sent to the fallback, i.e. it is passed
// to the Drop function.
func (channel MessageQueue) Push(msg *Message, timeout time.Duration) (state MessageState) {
	defer func() {
		// Treat closed channels like timeouts
		if recover() != nil {
			state = MessageStateTimeout
		}
	}()

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
				return MessageStateTimeout // ### return, fallback ###

			// Yield and try again
			default:
				spin.Yield()
			}
		}
	}
}

// IsEmpty returns true if no element is currently stored in the channel.
// Please note that this information can be extremely volatile in multithreaded
// environments.
func (channel MessageQueue) IsEmpty() bool {
	return len(channel) == 0
}

// GetNumQueued returns the number of queued messages.
// Please note that this information can be extremely volatile in multithreaded
// environments.
func (channel MessageQueue) GetNumQueued() int {
	return len(channel)
}

// PopWithTimeout returns a message from the buffer with a runtime <= maxDuration.
// If the channel is empty or the timout hit, the second return value is false.
func (channel MessageQueue) PopWithTimeout(maxDuration time.Duration) (*Message, bool) {
	timeout := time.NewTimer(maxDuration)
	select {
	case msg, more := <-channel:
		return msg, more
	case <-timeout.C:
		return nil, false
	}
}

// Pop returns a message from the buffer
func (channel MessageQueue) Pop() (*Message, bool) {
	msg, more := <-channel
	return msg, more
}

// Close stops the buffer from being able to recieve messages
func (channel MessageQueue) Close() {
	close(channel)
}
