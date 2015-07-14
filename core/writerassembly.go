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
	"github.com/trivago/gollum/core/log"
	"io"
)

// WriterAssembly is a helper struct for io.Writer compatible classes that use
// messagebatch.
type WriterAssembly struct {
	writer       io.Writer
	formatter    Formatter
	dropStreamID MessageStreamID
	buffer       []byte
}

// NewWriterAssembly creates a new adapter between io.Writer and the MessageBatch
// AssemblyFunc function signature
func NewWriterAssembly(writer io.Writer, formatter Formatter, dropStreamID MessageStreamID) WriterAssembly {
	return WriterAssembly{
		writer:       writer,
		formatter:    formatter,
		dropStreamID: dropStreamID,
	}
}

// SetWriter changes the writer interface used during Assemble
func (asm *WriterAssembly) SetWriter(writer io.Writer) {
	asm.writer = writer
}

// Write is an AssemblyFunc compatible implementation to pass all messages from
// a MessageBatch to an io.Writer.
// Messages are formatted using a given formatter. If the io.Writer fails to
// write the assembled buffer all messages are routed to the given drop channel.
func (asm *WriterAssembly) Write(messages []Message) {
	contentLen := 0
	for _, msg := range messages {
		payload, _ := asm.formatter.Format(msg)

		if contentLen+len(payload) > len(asm.buffer) {
			asm.buffer = append(asm.buffer[:contentLen], payload...)
		} else {
			copy(asm.buffer[contentLen:], payload)
			contentLen += len(payload)
		}
	}

	// Drop all messages if they could not be written to disk
	if _, err := asm.writer.Write(asm.buffer[:contentLen]); err != nil {
		Log.Error.Print("Stream write error:", err)
		asm.Drop(messages)
	}
}

// Drop is an AssemblyFunc compatible implementation to drop all messages from
// a MessageBatch.
func (asm *WriterAssembly) Drop(messages []Message) {
	for _, msg := range messages {
		msg.Route(asm.dropStreamID)
	}
}
