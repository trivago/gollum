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

package core

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"testing"
	"time"
)

type messageBatchWriter struct {
	expect  shared.Expect
	counter int
}

type mockFormatter struct {
}

func (bw *messageBatchWriter) hasData(messages []Message) {
	bw.expect.Greater(len(messages), 0)
}

func (bw *messageBatchWriter) checkOrder(messages []Message) {
	for i, msg := range messages {
		bw.expect.Equal(uint64(i), msg.Sequence)
	}
}

func (bw messageBatchWriter) Write(data []byte) (int, error) {
	bw.expect.Equal("0123456789", string(data))
	return len(data), nil
}

func (bw messageBatchWriter) Flush(msg Message) {
	bw.expect.NotExecuted()
}

func (bw *messageBatchWriter) Count(msg Message) {
	bw.counter++
}

func (mock mockFormatter) Format(msg Message) ([]byte, MessageStreamID) {
	return msg.Data, msg.StreamID
}

func TestMessageBatch(t *testing.T) {
	expect := shared.NewExpect(t)
	writer := messageBatchWriter{expect, 0}
	assembly := NewWriterAssembly(writer, writer.Flush, mockFormatter{})

	batch := NewMessageBatch(10)
	expect.False(batch.IsClosed())
	expect.True(batch.IsEmpty())

	// Append adds an item
	batch.Append(NewMessage(nil, []byte("test"), 0))
	expect.False(batch.IsEmpty())

	// Flush removes all items
	batch.Flush(writer.hasData)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.False(batch.ReachedSizeThreshold(10))

	// Append fails if buffer is full
	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}
	expect.False(batch.Append(NewMessage(nil, []byte("10"), 10)))
	expect.True(batch.ReachedSizeThreshold(10))

	// Test writer assembly
	batch.Flush(assembly.Write)
	batch.WaitForFlush(time.Second)
	expect.True(batch.IsEmpty())

	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}

	writer.counter = 0
	assembly.SetFlush(writer.Count)

	batch.Flush(assembly.Flush)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.Equal(10, writer.counter)

	// Flush removes all items, also if closed
	for i := 0; i < 5; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}

	batch.Close()
	expect.False(batch.Append(NewMessage(nil, []byte("6"), 6)))

	batch.Flush(writer.checkOrder)
	batch.WaitForFlush(time.Second)
	expect.True(batch.IsEmpty())

	expect.False(batch.Append(NewMessage(nil, nil, 0)))
	expect.False(batch.AppendOrBlock(NewMessage(nil, nil, 0)))
}

func TestMessageSerialize(t *testing.T) {
	expect := shared.NewExpect(t)
	now := time.Now()

	testMessage := Message{
		StreamID:     1,
		PrevStreamID: 2,
		Timestamp:    now,
		Sequence:     4,
		Data:         []byte("This is a\nteststring"),
	}

	data := testMessage.Serialize()
	expect.Greater(len(data), 0)

	readMessage, err := DeserializeMessage(data)
	expect.Nil(err)

	expect.Equal(readMessage.StreamID, testMessage.StreamID)
	expect.Equal(readMessage.PrevStreamID, testMessage.PrevStreamID)
	expect.Equal(readMessage.Timestamp, testMessage.Timestamp)
	expect.Equal(readMessage.Sequence, testMessage.Sequence)
	expect.Equal(readMessage.Data, testMessage.Data)
}
