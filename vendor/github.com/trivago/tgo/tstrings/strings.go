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

package tstrings

import (
	"fmt"
	"math"
	"strings"
)

var simpleEscapeChars = strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")
var jsonEscapeChars = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")

// Unescape replaces occurrences of \\n, \\r and \\t with real escape codes.
func Unescape(text string) string {
	return simpleEscapeChars.Replace(text)
}

// EscapeJSON replaces occurrences of \ and " with escaped versions.
func EscapeJSON(text string) string {
	return jsonEscapeChars.Replace(text)
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
		return fmt.Errorf("Number too large for buffer")
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

	return fmt.Errorf("Number too large for buffer")
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
		parsed := uint64(buffer[index] - '0')
		number = number*10 + parsed
		index++
	}

	return number, index
}

// IndexN returns the nth occurrences of sep in s or -1 if there is no nth
// occurrence of sep.
func IndexN(s, sep string, n int) int {
	sepIdx := 0
	for i := 0; i < n; i++ {
		nextIdx := strings.Index(s[sepIdx:], sep)
		if nextIdx == -1 {
			return -1 // ### return, not found ###
		}
		sepIdx += nextIdx + 1
	}
	return sepIdx - 1
}

// LastIndexN returns the nth occurrence of sep in s or -1 if there is no nth
// occurrence of sep. Searching is going from the end of the string to the start.
func LastIndexN(s, sep string, n int) int {
	if n == 0 {
		return -1 // ### return, nonsense ###
	}
	sepIdx := len(s)
	for i := 0; i < n; i++ {
		sepIdx = strings.LastIndex(s[:sepIdx], sep)
		if sepIdx == -1 {
			return -1 // ### return, not found ###
		}
	}
	return sepIdx
}
