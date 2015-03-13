package shared

import (
	"fmt"
	"io"
	"testing"
)

type StreamBufferWriter struct {
	expect          Expect
	successCalled   *bool
	errorCalled     *bool
	returnError     bool
	returnWrongSize bool
}

type mockFormatter struct {
	message string
}

func (writer StreamBufferWriter) Write(data []byte) (int, error) {
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

func (writer StreamBufferWriter) onSuccess() bool {
	*writer.successCalled = true
	return true
}

func (writer StreamBufferWriter) onError(err error) bool {
	*writer.errorCalled = true
	return false
}

func (mock *mockFormatter) PrepareMessage(msg Message) {
	mock.message = string(msg.Data)
}

func (mock *mockFormatter) Len() int {
	return len(mock.message)
}

func (mock *mockFormatter) String() string {
	return mock.message
}

func (mock *mockFormatter) Read(dest []byte) (int, error) {
	return copy(dest, []byte(mock.message)), nil
}

func (mock *mockFormatter) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write([]byte(mock.message))
	return int64(len), err
}

func TestStreamBuffer(t *testing.T) {
	expect := NewExpect(t)
	writer := StreamBufferWriter{expect, new(bool), new(bool), false, false}

	test10 := NewMessage(nil, []byte("1234567890"), []MessageStreamID{WildcardStreamID}, 0)
	test20 := NewMessage(nil, []byte("12345678901234567890"), []MessageStreamID{WildcardStreamID}, 1)
	buffer := NewStreamBuffer(15, new(mockFormatter))

	// Test optionals

	buffer.Flush(writer, nil, nil)
	buffer.WaitForFlush()

	// Test empty flush

	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush()

	expect.False(*writer.successCalled)
	expect.False(*writer.errorCalled)

	// Test regular appends

	result := buffer.Append(test10)
	expect.True(result)

	result = buffer.Append(test10)
	expect.False(result) // too large

	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush()

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
	buffer.WaitForFlush()

	expect.False(*writer.successCalled)
	expect.True(*writer.errorCalled)

	// Test writer size mismatch

	writer.returnWrongSize = true
	buffer.Flush(writer, writer.onSuccess, writer.onError)
	buffer.WaitForFlush()

	expect.False(*writer.successCalled)
	expect.True(*writer.errorCalled)
}
