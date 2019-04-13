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
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

func getMockMessage(data string) *Message {
	msg := &Message{
		prevStreamID: 2,
		source:       nil,
		timestamp:    time.Now().UnixNano(),
	}

	msg.data.payload = []byte(data)
	msg.streamID = 1

	return msg
}

func TestMessageEnqueue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for Enqueue()"
	msg := getMockMessage(msgString)
	buffer := NewMessageQueue(0)

	expect.Equal(MessageQueueDiscard, buffer.Push(msg, -1))

	go func() {
		expect.Equal(MessageQueueOk, buffer.Push(msg, 0))
	}()

	retMsg, _ := buffer.Pop()
	expect.Equal(msgString, retMsg.String())

	retStatus := buffer.Push(msg, 10*time.Millisecond)
	expect.Equal(MessageQueueTimeout, retStatus)

	go func() {
		expect.Equal(MessageQueueOk, buffer.Push(msg, 1*time.Second))
	}()

	retMsg, _ = buffer.Pop()
	expect.Equal(msgString, retMsg.String())
}

func TestMessageInstantiate(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for instantiate"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.FreezeOriginal()

	expect.Equal(msgString, string(msg.data.payload))
	expect.Equal(MessageStreamID(1), msg.streamID)
	expect.Equal(msgString, string(msg.orig.payload))
	expect.Equal(MessageStreamID(1), msg.origStreamID)
}

func TestMessageOriginalDataIntegrity(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for original data integrity"
	msgUpdateString := "Test for original data integrity - UPDATE"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.FreezeOriginal()

	msg.SetStreamID(MessageStreamID(10))
	msg.StorePayload([]byte(msgUpdateString))

	expect.Equal(msgUpdateString, string(msg.data.payload))
	expect.Equal(MessageStreamID(10), msg.streamID)
	expect.Equal(msgString, string(msg.orig.payload))
	expect.Equal(MessageStreamID(1), msg.origStreamID)
	expect.Equal(MessageStreamID(1), msg.prevStreamID)
}

func TestMessageClone(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for clone"
	msgUpdateString := "Test for clone - UPDATE"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.FreezeOriginal()

	msg.SetStreamID(MessageStreamID(10))
	msg.StorePayload([]byte(msgUpdateString))

	msgClone := msg.Clone()

	expect.Equal(msgUpdateString, string(msgClone.data.payload))
	expect.Equal(MessageStreamID(10), msgClone.streamID)
	expect.Equal(msgString, string(msgClone.orig.payload))
	expect.Equal(MessageStreamID(1), msgClone.origStreamID)
}

func TestMessageCloneMetadata(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for clone"
	msgUpdateString := "Test for clone - UPDATE"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.SetStreamID(MessageStreamID(10))
	msg.StorePayload([]byte(msgUpdateString))
	msg.GetMetadata().SetValue("foo", []byte("bar"))

	msg.Clone()

	expect.Equal("bar", msg.GetMetadata().GetValueString("foo"))
}

func TestMessageCloneOriginal(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for clone original"
	msgUpdateString := "Test for clone original - UPDATE"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.FreezeOriginal()

	msg.SetStreamID(MessageStreamID(10))
	msg.StorePayload([]byte(msgUpdateString))

	msgClone := msg.CloneOriginal()

	expect.Equal(msgString, string(msgClone.data.payload))
	expect.Equal(MessageStreamID(1), msgClone.streamID)
	expect.Equal(msgString, string(msgClone.orig.payload))
	expect.Equal(MessageStreamID(1), msgClone.origStreamID)
}

func TestMessageCloneOriginalMetadata(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for clone original"
	msgUpdateString := "Test for clone original - UPDATE"

	msg := NewMessage(nil, []byte(msgString), nil, 1)
	msg.FreezeOriginal()

	msg.SetStreamID(MessageStreamID(10))
	msg.StorePayload([]byte(msgUpdateString))
	msg.GetMetadata().SetValue("foo", []byte("bar"))

	clone := msg.CloneOriginal()

	// We froze before adding metadata, i.e. original metadata must be empty
	expect.Equal("", clone.GetMetadata().GetValueString("foo"))
	expect.Equal("bar", msg.GetMetadata().GetValueString("foo"))
}

func TestMessageMetadata(t *testing.T) {
	expect := ttesting.NewExpect(t)

	msg := NewMessage(nil, []byte("message payload"), nil, 1)
	value1 := []byte("value string")
	value2 := []byte("100")

	msg.GetMetadata().SetValue("key1", value1)
	msg.GetMetadata().SetValue("key2", value2)

	result1 := msg.GetMetadata().GetValue("key1")
	result2 := msg.GetMetadata().GetValue("key2")

	expect.Equal("value string", string(result1))
	expect.Equal("100", string(result2))
}

func TestMessageMetadataReset(t *testing.T) {
	expect := ttesting.NewExpect(t)

	msg := NewMessage(nil, []byte("message payload"), nil, 1)
	value := []byte("value string")

	msg.GetMetadata().SetValue("key1", value)

	result1 := msg.GetMetadata().GetValue("key1")

	msg.GetMetadata().Delete("key1")
	result2 := msg.GetMetadata().GetValue("key1")

	expect.Equal("value string", string(result1))
	expect.Equal([]byte{}, result2)
}

func TestMessageSerialize(t *testing.T) {
	expect := ttesting.NewExpect(t)
	testMessage := NewMessage(nil, []byte("This is a\nteststring"), nil, 1)
	testMessage.GetMetadata().SetValue("key", []byte("meta data value"))

	data, err := testMessage.Serialize()
	expect.NoError(err)
	expect.Greater(len(data), 0)

	// Test deserialization
	readMessage, err := DeserializeMessage(data)
	expect.NoError(err)

	expect.Equal(readMessage.streamID, testMessage.streamID)
	expect.Equal(readMessage.prevStreamID, testMessage.prevStreamID)
	expect.Equal(readMessage.origStreamID, testMessage.origStreamID)
	expect.Equal(readMessage.timestamp, testMessage.timestamp)
	expect.Equal(readMessage.data.payload, testMessage.data.payload)
	expect.Equal(readMessage.data.metadata, testMessage.data.metadata)
	expect.Nil(readMessage.orig)

	// Test original data serialization
	testMessage.FreezeOriginal()
	testMessage.SetlStreamIDAsOriginal(1)

	data, err = testMessage.Serialize()
	expect.NoError(err)
	expect.Greater(len(data), 0)

	readMessage, err = DeserializeMessage(data)
	expect.NoError(err)

	expect.Equal(readMessage.streamID, testMessage.streamID)
	expect.Equal(readMessage.prevStreamID, testMessage.prevStreamID)
	expect.Equal(readMessage.origStreamID, testMessage.origStreamID)
	expect.Equal(readMessage.timestamp, testMessage.timestamp)
	expect.Equal(readMessage.data.payload, testMessage.data.payload)
	expect.Equal(readMessage.data.metadata, testMessage.data.metadata)
	expect.NotNil(readMessage.orig)
	expect.Equal(readMessage.orig.payload, testMessage.orig.payload)
	expect.Equal(readMessage.orig.metadata, testMessage.orig.metadata)
}
