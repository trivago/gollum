// Copyright 2015-2016 trivago GmbH
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

package librdkafka

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/opt/librdkafka/lib -L/usr/local/lib -lrdkafka
// #include "wrapper.h"
import "C"

import (
	"unsafe"
)

// Message is the interface used to send exchange data with kafka
type Message interface {
	GetKey() []byte
	GetPayload() []byte
}

// SimpleMessage basic implementation of the Message interface
type SimpleMessage struct {
	key      []byte
	value    []byte
	metadata interface{}
}

// NewSimpleMessage creates a wrapper for a simple key/value pair. Use this
// strcut if you don't already have a container or your container cannot
// fulfill the Message interface.
func NewSimpleMessage(key []byte, value []byte, metadata interface{}) SimpleMessage {
	return SimpleMessage{
		key:      key,
		value:    value,
		metadata: metadata,
	}
}

// GetKey returns the (optional) key for this message
func (m SimpleMessage) GetKey() []byte {
	return m.key
}

// GetPayload returns the actual message data to be stored
func (m SimpleMessage) GetPayload() []byte {
	return m.value
}

// GetMetadata returns the custom data attached to this message
func (m SimpleMessage) GetMetadata() interface{} {
	return m.metadata
}

// PrepareBatch converts a message array to an array of native messages.
// The resulting pointer has to be freed with C.free(unsafe.Pointer(p)).
func PrepareBatch(messages []Message, topic *Topic, batchID uint64) *C.rd_kafka_message_t {
	topicID := C.int(topic.id)
	cBatchID := C.uint64_t(batchID)
	keyPtr := unsafe.Pointer(nil)
	valuePtr := unsafe.Pointer(nil)

	numMessages := C.int(len(messages))
	batch := C.CreateBatch(numMessages)

	for i := C.int(0); i < numMessages; i++ {
		msg := messages[i]
		key := msg.GetKey()
		value := msg.GetPayload()
		keyLen := C.int(len(key))
		valueLen := C.int(len(value))

		if keyLen > 0 {
			keyPtr = unsafe.Pointer(&key[0])
		} else {
			keyPtr = unsafe.Pointer(nil)
		}

		if valueLen > 0 {
			valuePtr = unsafe.Pointer(&value[0])
		} else {
			valuePtr = unsafe.Pointer(nil)
		}

		C.StoreBatchItem(batch, i, keyPtr, keyLen, valuePtr, valueLen, topicID, cBatchID)
	}

	return batch
}
