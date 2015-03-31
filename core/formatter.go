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
	"io"
)

const (
	// DefaultTimestamp is the timestamp format string used for messages
	DefaultTimestamp = "2006-01-02 15:04:05 MST | "
	// DefaultDelimiter is the default end of message delimiter
	DefaultDelimiter = "\n"
)

// Formatter is the interface definition for message formatters
type Formatter interface {
	fmt.Stringer // String() string
	io.Reader    // Read([]byte) (int, error)
	io.WriterTo  // WriteTo(io.Writer) (int64, error)

	// Len returns the length of a formatted message.
	Len() int

	// Bytes returns the message as a byte slice
	Bytes() []byte

	// PrepareMessage sets the message to be formatted. This allows the
	// formatter to build up caches for subsequent method calls.
	PrepareMessage(msg Message)
}

// FormatterBase provides basic functionality for all formatters that generate
// the final message in PrepareMessage.
type FormatterBase struct {
	Message []byte
}

// Len returns the length of a formatted message.
func (format *FormatterBase) Len() int {
	return len(format.Message)
}

// String returns the message as string
func (format *FormatterBase) String() string {
	return string(format.Message)
}

// Bytes returns the message as a byte slice
func (format *FormatterBase) Bytes() []byte {
	return format.Message
}

// Read copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *FormatterBase) Read(dest []byte) (int, error) {
	return copy(dest, format.Message), nil
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *FormatterBase) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write(format.Message)
	return int64(len), err
}
