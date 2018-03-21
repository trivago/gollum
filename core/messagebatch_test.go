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
	"fmt"
	"github.com/trivago/tgo/ttesting"
	"testing"
	"time"
)

type messageBatchWriter struct {
	expect  ttesting.Expect
	counter int
	data    []byte
}

func (bw *messageBatchWriter) hasData(messages []*Message) {
	bw.expect.Greater(len(messages), 0)
}

func (bw messageBatchWriter) Write(data []byte) (int, error) {
	bw.expect.Equal("0123456789", string(data))
	return len(data), nil
}

func (bw *messageBatchWriter) AppendData(data []byte) (int, error) {
	bw.data = append(bw.data, data...)
	return len(data), nil
}

func (bw messageBatchWriter) Flush(msg *Message) {
	bw.expect.NotExecuted()
}

func (bw *messageBatchWriter) Count(msg *Message) {
	bw.counter++
}

func TestMessageBatchAppendOrFlush(t *testing.T) {
	expect := ttesting.NewExpect(t)
	writer := messageBatchWriter{expect, 0, []byte{}}
	batch := NewMessageBatch(10)

	flushBuffer := func() {
		batch.Flush(func(a []*Message) {
			for _, msg := range a {
				writer.AppendData(msg.GetPayload())
			}
		})
		batch.WaitForFlush(time.Second)
		expect.True(batch.IsEmpty())
	}

	doBlock := func() bool {
		return true
	}

	dontBlock := func() bool {
		return false
	}
	//dropMsg a stub
	dropMsg := func(msg *Message) {
	}

	for i := 0; i < 10; i++ {
		batch.AppendOrFlush(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), nil, InvalidStreamID),
			flushBuffer,
			dontBlock,
			dropMsg)
	}
	// the buffer is full so it should be flushed and the new message queued
	batch.AppendOrFlush(NewMessage(nil, []byte(fmt.Sprintf("%d", 10)), nil, InvalidStreamID),
		flushBuffer,
		doBlock,
		dropMsg)

	expect.Equal(batch.getActiveBufferCount(), int(1))
	expect.Equal("0123456789", string(writer.data))

}

func TestMessageBatch(t *testing.T) {
	expect := ttesting.NewExpect(t)
	writer := messageBatchWriter{expect, 0, []byte{}}
	assembly := NewWriterAssembly(writer, writer.Flush, &mockFormatter{})

	batch := NewMessageBatch(10)
	expect.False(batch.IsClosed())
	expect.True(batch.IsEmpty())

	// length of buffer should be 10
	expect.Equal(batch.Len(), 10)

	// Append adds an item
	batch.Append(NewMessage(nil, []byte("test"), nil, InvalidStreamID))
	expect.False(batch.IsEmpty())

	// Flush removes all items
	batch.Flush(writer.hasData)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.False(batch.ReachedSizeThreshold(10))

	// Append fails if buffer is full
	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), nil, InvalidStreamID)))
	}
	expect.False(batch.Append(NewMessage(nil, []byte("10"), nil, InvalidStreamID)))
	expect.True(batch.ReachedSizeThreshold(10))

	// Test writer assembly
	batch.Flush(assembly.Write)
	batch.WaitForFlush(time.Second)
	expect.True(batch.IsEmpty())

	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), nil, InvalidStreamID)))
	}

	writer.counter = 0
	assembly.SetFlush(writer.Count)

	batch.Flush(assembly.Flush)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.Equal(10, writer.counter)

	// Flush removes all items, also if closed
	for i := 0; i < 5; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), nil, InvalidStreamID)))
	}

	batch.Flush(assembly.Flush)
	batch.WaitForFlush(time.Second)
	// batch is not closed and message is appended

	for i := 0; i < 10; i++ {
		expect.True(batch.AppendOrBlock(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), nil, InvalidStreamID)))
	}
	go func() {
		expect.False(batch.AppendOrBlock(NewMessage(nil, []byte("10"), nil, InvalidStreamID)))
	}()
	//let above goroutine run so that spin can Yield atleast once
	time.Sleep(1 * time.Second)

	expect.True(batch.ReachedTimeThreshold(100 * time.Millisecond))

	//closing now will close the messageBatch and the goroutine will return false
	batch.Close(func([]*Message) {}, time.Second)
	expect.True(batch.IsEmpty())
	expect.False(batch.Append(NewMessage(nil, []byte("6"), nil, InvalidStreamID)))
	expect.True(batch.IsEmpty())

	expect.False(batch.Append(NewMessage(nil, nil, nil, InvalidStreamID)))
	expect.False(batch.AppendOrBlock(NewMessage(nil, nil, nil, InvalidStreamID)))
}
