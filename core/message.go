// Copyright 2015-2017 trivago GmbH
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
	"github.com/golang/protobuf/proto"
	"github.com/trivago/tgo/tcontainer"
	"time"
)

// MessageState is used as a return value for the Enqueue method
type MessageState int

// MessageSource defines methods that are common to all message sources.
// Currently this is only a placeholder.
type MessageSource interface {
	// IsActive returns true if the source can produce messages
	IsActive() bool

	// IsBlocked returns true if the source cannot produce messages
	IsBlocked() bool
}

// AsyncMessageSource extends the MessageSource interface to allow a backchannel
// that simply forwards any message coming from the producer.
type AsyncMessageSource interface {
	MessageSource

	// EnqueueResponse sends a message to the source of another message.
	EnqueueResponse(msg *Message)
}

// SerialMessageSource extends the AsyncMessageSource interface to allow waiting
// for all parts of the response to be submitted.
type SerialMessageSource interface {
	AsyncMessageSource

	// Notify the end of the response stream
	ResponseDone()
}

// LinkableMessageSource extends the MessageSource interface to allow a pipe
// like behaviour between two components that communicate messages.
type LinkableMessageSource interface {
	MessageSource
	// Link the message source to the message receiver. This makes it possible
	// to create stable "pipes" between e.g. a consumer and producer.
	Link(pipe interface{})

	// IsLinked has to return true if Link executed successful and does not
	// need to be called again.
	IsLinked() bool
}

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	data         []byte
	streamID     MessageStreamID
	prevStreamID MessageStreamID
	source       MessageSource
	timestamp    time.Time
	sequence     uint64
}

var (
	// MessageDataPool is the pool used for message payloads.
	// This pool should be used to allocate temporary buffers for e.g.
	// formatters.
	MessageDataPool = tcontainer.NewBytePool()
)

// NewMessage creates a new message from a given data stream by copying data.
func NewMessage(source MessageSource, data []byte, sequence uint64, streamID MessageStreamID) *Message {
	buffer := MessageDataPool.Get(len(data))
	copy(buffer, data)

	return &Message{
		data:         buffer,
		source:       source,
		streamID:     streamID,
		prevStreamID: streamID,
		timestamp:    time.Now(),
		sequence:     sequence,
	}
}

// NewMessageWithSize creates a new message with a buffer of a given size.
// The buffer may contain data from previous messages.
func NewMessageWithSize(source MessageSource, dataSize int, sequence uint64, streamID MessageStreamID) *Message {
	return &Message{
		data:         MessageDataPool.Get(dataSize),
		source:       source,
		streamID:     streamID,
		prevStreamID: streamID,
		timestamp:    time.Now(),
		sequence:     sequence,
	}
}

// Created returns the time when this message was created.
func (msg *Message) Created() time.Time {
	return msg.timestamp
}

// Sequence returns the message's sequence number.
func (msg *Message) Sequence() uint64 {
	return msg.sequence
}

// StreamID returns the stream this message is currently routed to.
func (msg *Message) StreamID() MessageStreamID {
	return msg.streamID
}

// PreviousStreamID returns the last "hop" of this message.
func (msg *Message) PreviousStreamID() MessageStreamID {
	return msg.prevStreamID
}

// GetStream returns the stream object behind the current StreamID.
func (msg *Message) GetStream() Stream {
	return StreamRegistry.GetStreamOrFallback(msg.streamID)
}

// GetPreviousStream returns the stream object behind the previous StreamID.
func (msg *Message) GetPreviousStream() Stream {
	return StreamRegistry.GetStreamOrFallback(msg.prevStreamID)
}

// SetStreamID sets a new stream and stores the current one in the previous
// stream field.
func (msg *Message) SetStreamID(streamID MessageStreamID) {
	msg.prevStreamID = msg.streamID
	msg.streamID = streamID
}

// Source returns the message's source (can be nil).
func (msg *Message) Source() MessageSource {
	return msg.source
}

// String implements the stringer interface
func (msg *Message) String() string {
	return string(msg.data)
}

// Data returns the stored data
func (msg *Message) Data() []byte {
	return msg.data
}

// Len returns the length of the current data buffer
func (msg *Message) Len() int {
	return len(msg.data)
}

// Cap returns the capacity of the current data buffer
func (msg *Message) Cap() int {
	return cap(msg.data)
}

// Store copies data into the hold data buffer. If the buffer can hold data
// it is resized, otherwise a new buffer will be allocated.
func (msg *Message) Store(data []byte) {
	copy(msg.Resize(len(data)), data)
}

// Offset moves the slice start offset of the currently stored data to the
// given position. This can be used to e.g. efficiently crop of the beginning
// of a message.
func (msg *Message) Offset(offset int) {
	msg.data = msg.data[offset:]
}

// Resize changes the size of the stored buffer. The current content is not
// guaranteed to be preserved. If content needs to be preserved use Extend.
func (msg *Message) Resize(size int) []byte {
	switch {
	case size == len(msg.data):
	case size <= cap(msg.data):
		msg.data = msg.data[:size]
	default:
		msg.data = MessageDataPool.Get(size)
	}

	return msg.data
}

// Extend changes the size of the stored buffer. The current content will be
// preserved. If content does not need to be preserved use Resize.
func (msg *Message) Extend(size int) []byte {
	switch {
	case size == len(msg.data):
	case size <= cap(msg.data):
		msg.data = msg.data[:size]
	default:
		old := msg.data
		msg.data = MessageDataPool.Get(size)
		copy(msg.data, old)
	}

	return msg.data
}

// Clone returns a copy of this message, i.e. the payload is duplicated.
// The created timestamp is copied, too.
func (msg *Message) Clone() *Message {
	clone := *msg
	clone.data = MessageDataPool.Get(len(msg.data))
	copy(clone.data, msg.data)

	return &clone
}

// Serialize generates a string containing all data that can be preserved over
// shutdown (i.e. no data directly referencing runtime components).
func (msg Message) Serialize() ([]byte, error) {
	serializable := &SerializedMessage{
		StreamID:     proto.Uint64(uint64(msg.streamID)),
		PrevStreamID: proto.Uint64(uint64(msg.prevStreamID)),
		Timestamp:    proto.Int64(msg.timestamp.UnixNano()),
		Sequence:     proto.Uint64(msg.sequence),
		Data:         msg.data,
	}

	return proto.Marshal(serializable)
}

// DeserializeMessage generates a message from a string produced by
// Message.Serialize.
func DeserializeMessage(data []byte) (Message, error) {
	serializable := new(SerializedMessage)
	err := proto.Unmarshal(data, serializable)

	msg := Message{
		streamID:     MessageStreamID(serializable.GetStreamID()),
		prevStreamID: MessageStreamID(serializable.GetPrevStreamID()),
		timestamp:    time.Unix(0, serializable.GetTimestamp()),
		sequence:     serializable.GetSequence(),
		data:         serializable.GetData(),
	}

	return msg, err
}
