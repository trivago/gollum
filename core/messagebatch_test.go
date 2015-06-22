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

type MessageBatchWriter struct {
	expect          shared.Expect
	successCalled   *bool
	errorCalled     *bool
	returnError     bool
	returnWrongSize bool
}

type mockFormatter struct {
}

func (writer MessageBatchWriter) Write(data []byte) (int, error) {
	defer func() {
		*writer.successCalled = false
		*writer.errorCalled = false
	}()

	if writer.returnWrongSize {
		return 0, nil
	}

	if writer.returnError {
		return len(data), fmt.Errorf("test")
	}

	return len(data), nil
}

func (writer MessageBatchWriter) onSuccess() bool {
	*writer.successCalled = true
	return true
}

func (writer MessageBatchWriter) onError(err error) bool {
	*writer.errorCalled = true
	return false
}

func (mock *mockFormatter) Format(msg Message) ([]byte, MessageStreamID) {
	return msg.Data, msg.StreamID
}

func TestMessageBatch(t *testing.T) {
	expect := shared.NewExpect(t)
	writer := MessageBatchWriter{expect, new(bool), new(bool), false, false}

	test10 := NewMessage(nil, []byte("1234567890"), 0)
	test20 := NewMessage(nil, []byte("12345678901234567890"), 1)
	buffer := NewMessageBatch(15, new(mockFormatter))

	// Test optionals

	buffer.Flush(writer, nil, nil)
	buffer.WaitForFlush(time.Duration(0))

	// Test empty flush

	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush(time.Duration(0))

	expect.False(*writer.successCalled)
	expect.False(*writer.errorCalled)

	// Test regular appends

	result := buffer.Append(test10)
	expect.True(result)

	result = buffer.Append(test10)
	expect.False(result) // too large

	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush(time.Duration(0))

	expect.True(*writer.successCalled)
	expect.False(*writer.errorCalled)

	// Test oversize append

	result = buffer.Append(test20)
	expect.True(result) // Too large -> ignored

	// Test writer error

	result = buffer.Append(test10)
	expect.True(result)

	writer.returnError = true
	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush(time.Duration(0))

	expect.False(*writer.successCalled)
	expect.True(*writer.errorCalled)

	// Test writer size mismatch

	writer.returnWrongSize = true
	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush(time.Duration(0))

	expect.False(*writer.successCalled)
	expect.True(*writer.errorCalled)
}
