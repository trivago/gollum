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

// MessageData is a container for the message payload, streamID and an optional message key
// The struct is used by Message.data for the current message data and orig for the original message data
type MessageData struct {
	payload  []byte
	streamID MessageStreamID
	Metadata Metadata
}

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	data         MessageData
	orig         MessageData
	prevStreamID MessageStreamID
	source       MessageSource
	timestamp    time.Time
}

var (
	// MessageDataPool is the pool used for message payloads.
	// This pool should be used to allocate temporary buffers for e.g.
	// formatters.
	MessageDataPool = tcontainer.NewBytePoolWithSize(2)
)

// NewMessage creates a new message from a given data stream by copying data.
func NewMessage(source MessageSource, data []byte, metadata Metadata, streamID MessageStreamID) *Message {
	buffer := getPayloadCopy(data)

	message := &Message{
		source:       source,
		prevStreamID: streamID,
		timestamp:    time.Now(),
	}

	message.data.payload = buffer
	message.data.streamID = streamID
	if metadata == nil {
		message.data.Metadata = make(Metadata)
	} else {
		message.data.Metadata = metadata
	}

	return message
}

// getPayloadCopy return a copy of the data byte array
func getPayloadCopy(data []byte) (buffer []byte) {
	buffer = MessageDataPool.Get(len(data))
	copy(buffer, data)
	return
}

// GetCreationTime returns the time when this message was created.
func (msg *Message) GetCreationTime() time.Time {
	return msg.timestamp
}

// GetStreamID returns the stream this message is currently routed to.
func (msg *Message) GetStreamID() MessageStreamID {
	return msg.data.streamID
}

// GetOrigStreamID returns the original/first streamID
func (msg *Message) GetOrigStreamID() MessageStreamID {
	return msg.orig.streamID
}

// GetPrevStreamID returns the last "hop" of this message.
func (msg *Message) GetPrevStreamID() MessageStreamID {
	return msg.prevStreamID
}

// GetRouter returns the stream object behind the current StreamID.
func (msg *Message) GetRouter() Router {
	return StreamRegistry.GetRouterOrFallback(msg.GetStreamID())
}

// GetPrevRouter returns the stream object behind the previous StreamID.
func (msg *Message) GetPrevRouter() Router {
	return StreamRegistry.GetRouterOrFallback(msg.prevStreamID)
}

// SetStreamID sets a new stream and stores the current one in the previous
// stream field.
func (msg *Message) SetStreamID(streamID MessageStreamID) {
	msg.prevStreamID = msg.GetStreamID()
	msg.data.streamID = streamID
}

// GetSource returns the message's source (can be nil).
func (msg *Message) GetSource() MessageSource {
	return msg.source
}

// String implements the stringer interface
func (msg *Message) String() string {
	return string(msg.data.payload)
}

// GetPayload returns the stored data
func (msg *Message) GetPayload() []byte {
	return msg.data.payload
}

// GetMetadata returns the current Metadata
func (msg *Message) GetMetadata() Metadata {
	return msg.data.Metadata
}

// StorePayload copies data into the hold data buffer. If the buffer can hold
// data it is resized, otherwise a new buffer will be allocated.
func (msg *Message) StorePayload(data []byte) {
	copy(msg.ResizePayload(len(data)), data)
}

// ResizePayload changes the size of the stored buffer. The current content is
// not guaranteed to be preserved. If content needs to be preserved use Extend.
func (msg *Message) ResizePayload(size int) []byte {
	switch {
	case size == len(msg.data.payload):
	case size <= cap(msg.data.payload):
		msg.data.payload = msg.data.payload[:size]
	default:
		msg.data.payload = MessageDataPool.Get(size)
	}

	return msg.data.payload
}

// ExtendPayload changes the size of the stored buffer. The current content will
// be preserved. If content does not need to be preserved use Resize.
func (msg *Message) ExtendPayload(size int) []byte {
	switch {
	case size == len(msg.data.payload):
	case size <= cap(msg.data.payload):
		msg.data.payload = msg.data.payload[:size]
	default:
		old := msg.data.payload
		msg.data.payload = MessageDataPool.Get(size)
		copy(msg.data.payload, old)
	}

	return msg.data.payload
}

// Clone returns a copy of this message, i.e. the payload is duplicated.
// The created timestamp is copied, too.
func (msg *Message) Clone() *Message {
	clone := *msg

	clone.data.payload = MessageDataPool.Get(len(msg.data.payload))
	copy(clone.data.payload, msg.data.payload)

	return &clone
}

// CloneOriginal returns a copy of this message with the original payload and stream
// The created timestamp is copied, too.
func (msg *Message) CloneOriginal() *Message {
	clone := *msg

	clone.data.payload = MessageDataPool.Get(len(msg.orig.payload))
	copy(clone.data.payload, msg.orig.payload)

	clone.SetStreamID(msg.orig.streamID)

	return &clone
}

// FreezeOriginal set the original data and freeze the message
func (msg *Message) FreezeOriginal() {
	msg.orig.payload = getPayloadCopy(msg.data.payload)
	msg.orig.streamID = msg.data.streamID
	msg.orig.Metadata = msg.data.Metadata.Clone()
}

// Serialize generates a new payload containing all data that can be preserved
// over shutdown (i.e. no data directly referencing runtime components). The
// serialized data is based on the current message state.
func (msg *Message) Serialize() ([]byte, error) {
	return msg.data.serialize(msg.prevStreamID, msg.timestamp)
}

// SerializeOriginal generates a new payload containing all data that can be
// preserved over shutdown (i.e. no data directly referencing runtime c
// omponents). The serialized data is based on the original message
func (msg *Message) SerializeOriginal() ([]byte, error) {
	return msg.orig.serialize(msg.data.streamID, msg.timestamp)
}

func (data MessageData) serialize(prevStreamID MessageStreamID, timestamp time.Time) ([]byte, error) {
	serializable := &SerializedMessage{
		StreamID:     proto.Uint64(uint64(data.streamID)),
		PrevStreamID: proto.Uint64(uint64(prevStreamID)),
		Timestamp:    proto.Int64(timestamp.UnixNano()),
		Data:         data.payload,
		Metadata:     data.Metadata,
	}

	return proto.Marshal(serializable)
}

// DeserializeMessage generates a message from a string produced by
// Message.Serialize.
func DeserializeMessage(data []byte) (Message, error) {
	serializable := new(SerializedMessage)
	err := proto.Unmarshal(data, serializable)

	msg := Message{
		prevStreamID: MessageStreamID(serializable.GetPrevStreamID()),
		timestamp:    time.Unix(0, serializable.GetTimestamp()),
	}

	msg.data.streamID = MessageStreamID(serializable.GetStreamID())
	msg.data.payload = serializable.GetData()
	msg.data.Metadata = serializable.GetMetadata()

	msg.FreezeOriginal()
	return msg, err
}
