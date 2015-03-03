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

package shared

import (
	"errors"
	"math"
)

const (
	// DefaultTimestamp is the timestamp format string used for messages
	DefaultTimestamp = "2006-01-02 15:04:05 MST | "
	// DefaultDelimiter is the default end of message delimiter
	DefaultDelimiter = "\n"
)

// Formatter is the interface definition for message formatters
type Formatter interface {
	// PrepareMessage sets the message to be formatted. This allows the
	// formatter to build up caches for subsequent method calls.
	PrepareMessage(msg Message)

	// GetLength returns the length of a formatted message returned by String()
	// or CopyTo().
	GetLength() int

	// String returns the message as string
	String() string

	// CopyTo copies the message into an existing buffer. It is assumed that
	// dest has enough space to fit GetLength() bytes
	CopyTo(dest []byte) int
}

// ItoLen returns the length of an unsingned integer when converted to a string
func ItoLen(number uint64) int {
	switch {
	case number < 10:
		return 1
	default:
		return int(math.Log10(float64(number)) + 1)
	}
}

// Itob writes an unsigned integer to the start of a given byte buffer.
func Itob(number uint64, buffer []byte) error {
	numberLen := ItoLen(number)
	bufferLen := len(buffer)

	if numberLen > bufferLen {
		return errors.New("Number too large for buffer")
	}

	for i := numberLen - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10
	}

	return nil
}

// Itobe writes an unsigned integer to the end of a given byte buffer.
func Itobe(number uint64, buffer []byte) error {
	for i := len(buffer) - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10

		// Check here because the number 0 has to be written, too
		if number == 0 {
			return nil
		}
	}

	return errors.New("Number too large for buffer")
}

// Btoi is a fast byte buffer to unsigned int parser that reads until the first
// non-number character is found. It returns the number value as well as the
// length of the number string encountered.
// If a number could not be parsed the returned length will be 0
func Btoi(buffer []byte) (uint64, int) {
	number := uint64(0)
	index := 0
	bufferLen := len(buffer)

	for index < bufferLen && buffer[index] >= '0' && buffer[index] <= '9' {
		number = number*10 + uint64(buffer[index]-'0')
		index++
	}

	return number, index
}
