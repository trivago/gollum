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

	// PrepareMessage sets the message to be formatted. This allows the
	// formatter to build up caches for subsequent method calls.
	PrepareMessage(msg Message)

	// Len returns the length of a formatted message.
	Len() int
}
