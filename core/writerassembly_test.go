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
	"errors"
	"testing"

	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type mockIoWrite struct {
	e ttesting.Expect
}

// created to check the change of writer
type secondMockIoWrite struct {
}

func (iw mockIoWrite) Write(data []byte) (n int, err error) {
	return len(data), nil
}

func (iw secondMockIoWrite) Write(data []byte) (n int, err error) {
	return len(data), errors.New("someError")
}

func (iw mockIoWrite) mockFlush(m *Message) {
	iw.e.Equal(m.String(), "abcde")
}

func TestWriterAssemblySetValidator(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockIo := mockIoWrite{expect}
	wa := NewWriterAssembly(mockIo, mockIo.mockFlush, &mockFormatter{})
	validator := func() bool {
		return true
	}
	wa.SetValidator(validator)
	expect.True(wa.validate())
}

func TestWriterAssemblySetErrorHandler(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockIo := mockIoWrite{expect}

	// Was tlog.SetCacheWriter() previously. Needed?
	logrus.SetOutput(ioutil.Discard)

	wa := NewWriterAssembly(mockIo, mockIo.mockFlush, &mockFormatter{})
	handler := func(e error) bool {
		return e.Error() == "abc"
	}

	wa.SetErrorHandler(handler)
	expect.True(wa.handleError(errors.New("abc")))
}

func TestWriterAssemblySetWriter(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockIo := mockIoWrite{expect}
	wa := NewWriterAssembly(mockIo, mockIo.mockFlush, &mockFormatter{})

	wa.SetWriter(secondMockIoWrite{})
	length, _ := wa.writer.Write([]byte("abcde"))
	expect.Equal(length, 5)
}

func TestWriterAssemblyWrite(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockIo := mockIoWrite{expect}
	wa := NewWriterAssembly(nil, mockIo.mockFlush, &mockFormatter{})

	msg1 := NewMessage(nil, []byte("abcde"), nil, InvalidStreamID)

	// should give error msg and flush msg as there writer is not available yet
	// test here is done in mockIo.mockFlush
	wa.Write([]*Message{msg1})

	wa.SetWriter(mockIo)
	//this should extend the wa's buffer to 5 -> "abcde"
	wa.Write([]*Message{msg1})
	//should override in the same buffer from beginning
	wa.Write([]*Message{msg1})

	//set validator to return false so final flush is executed
	validator := func() bool {
		return false
	}
	wa.SetValidator(validator)
	wa.Write([]*Message{msg1})
	// this writer object return error value
	wa.SetWriter(secondMockIoWrite{})
	wa.Write([]*Message{msg1})

	// set errorHandler
	handlerReturnsTrue := func(e error) bool {
		return e.Error() == "someError"
	}
	wa.SetErrorHandler(handlerReturnsTrue)
	wa.Write([]*Message{msg1})

	handlerWithoutError := func(e error) bool {
		return e.Error() != "someError"
	}
	wa.SetErrorHandler(handlerWithoutError)
	wa.Write([]*Message{msg1})

}
