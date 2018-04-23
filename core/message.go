// Copyright 2015-2018 trivago N.V.
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
	"time"

	"github.com/golang/protobuf/proto"
)

// MessageData is a container for the message payload, streamID and an optional message key
// The struct is used by Message.data for the current message data and orig for the original message data
type MessageData struct {
	payload  []byte
	metadata Metadata
}

// Message is a container used for storing the internal state of messages.
// This struct is passed between consumers and producers.
type Message struct {
	data         MessageData
	orig         *MessageData
	streamID     MessageStreamID
	prevStreamID MessageStreamID
	origStreamID MessageStreamID
	source       MessageSource
	timestamp    time.Time
}

// NewMessage creates a new message from a given data stream by copying data.
func NewMessage(source MessageSource, data []byte, metadata Metadata, streamID MessageStreamID) *Message {
	msg := &Message{
		source:       source,
		streamID:     streamID,
		origStreamID: streamID,
		timestamp:    time.Now(),
	}

	msg.data.payload = getPayloadCopy(data)
	if metadata != nil && len(metadata) > 0 {
		msg.data.metadata = metadata
	}

	return msg
}

// getPayloadCopy return a copy of the data byte array
func getPayloadCopy(data []byte) (buffer []byte) {
	buffer = make([]byte, len(data))
	copy(buffer, data)
	return
}

// GetCreationTime returns the time when this message was created.
func (msg *Message) GetCreationTime() time.Time {
	return msg.timestamp
}

// GetStreamID returns the stream this message is currently routed to.
func (msg *Message) GetStreamID() MessageStreamID {
	return msg.streamID
}

// GetOrigStreamID returns the original/first streamID
func (msg *Message) GetOrigStreamID() MessageStreamID {
	return msg.origStreamID
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
	return StreamRegistry.GetRouterOrFallback(msg.GetPrevStreamID())
}

// GetOrigRouter returns the stream object behind the original StreamID.
func (msg *Message) GetOrigRouter() Router {
	return StreamRegistry.GetRouterOrFallback(msg.GetOrigStreamID())
}

// SetStreamID sets a new stream and stores the current one in the previous
// stream field. This method does not affect the original stream ID.
func (msg *Message) SetStreamID(streamID MessageStreamID) {
	msg.prevStreamID = msg.streamID
	msg.streamID = streamID
}

// SetlStreamIDAsOriginal acts like SetStreamID but always sets the original
// stream ID, too. This method should be used before a message is routed for the first
// time (e.g. in a consumer) or when the original stream should change.
func (msg *Message) SetlStreamIDAsOriginal(streamID MessageStreamID) {
	msg.SetStreamID(streamID)
	msg.origStreamID = streamID
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

// GetMetadata returns the current Metadata. If no metadata is present, the
// metadata map will be created by this call.
func (msg *Message) GetMetadata() Metadata {
	if msg.data.metadata == nil {
		msg.data.metadata = make(Metadata)
	}
	return msg.data.metadata
}

// TryGetMetadata returns the current Metadata. If no metadata is present, nil
// will be returned.
func (msg *Message) TryGetMetadata() Metadata {
	return msg.data.metadata
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
		msg.data.payload = make([]byte, size)
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
		msg.data.payload = make([]byte, size)
		copy(msg.data.payload, old)
	}

	return msg.data.payload
}

// Clone returns a copy of this message, i.e. the payload is duplicated.
// The created timestamp is copied, too.
func (msg *Message) Clone() *Message {
	clone := *msg

	clone.data.payload = make([]byte, len(msg.data.payload))
	copy(clone.data.payload, msg.data.payload)

	return &clone
}

// CloneOriginal returns a copy of this message with the original payload and
// stream. If FreezeOriginal has not been called before it will be at this point
// so that all subsequential calls will use the same original.
func (msg *Message) CloneOriginal() *Message {
	if msg.orig == nil {
		msg.FreezeOriginal()
	}

	clone := *msg
	clone.data.payload = make([]byte, len(msg.orig.payload))
	copy(clone.data.payload, msg.orig.payload)

	if msg.orig.metadata == nil {
		clone.data.metadata = msg.orig.metadata.Clone()
	} else {
		clone.data.metadata = nil
	}

	clone.SetStreamID(msg.origStreamID)
	return &clone
}

// FreezeOriginal will take the current state of the message and store it as
// the "original" message. This function can only be called once for each
// message. Please note that this function only affects payload and metadata and
// can only be called once. Additional calls will have no effect.
// The original stream ID can be changed at any time by using SetOrigStreamID.
func (msg *Message) FreezeOriginal() {
	// Freeze may be called only once
	if msg.orig != nil {
		return
	}

	var metadata Metadata
	if msg.data.metadata != nil {
		metadata = msg.data.metadata.Clone()
	}

	msg.orig = &MessageData{
		payload:  getPayloadCopy(msg.data.payload),
		metadata: metadata,
	}
}

// Serialize generates a new payload containing all data that can be preserved
// over shutdown (i.e. no data directly referencing runtime components). The
// serialized data is based on the current message state and does not preserve
// the original data created by FreezeOriginal.
func (msg *Message) Serialize() ([]byte, error) {
	serializable := &SerializedMessage{
		StreamID:     proto.Uint64(uint64(msg.GetStreamID())),
		PrevStreamID: proto.Uint64(uint64(msg.GetPrevStreamID())),
		OrigStreamID: proto.Uint64(uint64(msg.GetOrigStreamID())),
		Timestamp:    proto.Int64(msg.timestamp.UnixNano()),
		Data: &SerializedMessageData{
			Data:     msg.data.payload,
			Metadata: msg.data.metadata,
		},
	}

	if msg.orig != nil {
		serializable.Original = &SerializedMessageData{
			Data:     msg.orig.payload,
			Metadata: msg.orig.metadata,
		}
	}

	return proto.Marshal(serializable)
}

// DeserializeMessage generates a message from a byte array produced by
// Message.Serialize. Please note that the payload is restored but the original
// data is not. As of this FreezeOriginal can be called again after this call.
func DeserializeMessage(data []byte) (*Message, error) {
	serializable := new(SerializedMessage)
	if err := proto.Unmarshal(data, serializable); err != nil {
		return nil, err
	}

	timestamp := serializable.GetTimestamp()
	timestampSec := timestamp / 1e9
	timestampNano := timestamp - timestampSec*1e9

	msg := &Message{
		streamID:     MessageStreamID(serializable.GetStreamID()),
		prevStreamID: MessageStreamID(serializable.GetPrevStreamID()),
		origStreamID: MessageStreamID(serializable.GetOrigStreamID()),
		timestamp:    time.Unix(timestampSec, timestampNano),
	}

	if msgData := serializable.GetData(); msgData != nil {
		msg.data.payload = msgData.GetData()
		msg.data.metadata = msgData.GetMetadata()
	}

	if msgOrigData := serializable.GetOriginal(); msgOrigData != nil {
		msg.orig = new(MessageData)
		msg.orig.payload = msgOrigData.GetData()
		msg.orig.metadata = msgOrigData.GetMetadata()
	}

	return msg, nil
}
