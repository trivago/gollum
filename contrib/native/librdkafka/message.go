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

// #cgo CFLAGS: -I/usr/local/include -std=c99
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
	GetUserdata() []byte
}

// MessageDelivery is used to handle message delivery errors
type MessageDelivery interface {
	OnMessageError(reason string, userdata []byte)
}

//export goDeliveryHandler
func goDeliveryHandler(clientHandle *C.rd_kafka_t, code C.int, bufferPtr *C.buffer_t) {
	if handler, exists := clients[clientHandle]; exists {
		buffer := UnmarshalBuffer(bufferPtr)
		handler.OnMessageError(codeToString(int(code)), buffer)
	}
}

// UnmarshalBuffer creates a byte slice copy from a buffer_t handle.
func UnmarshalBuffer(bufferPtr *C.buffer_t) []byte {
	length := int(bufferPtr.len)
	buffer := make([]byte, length)
	copy(buffer, (*[1 << 30]byte)(bufferPtr.data)[:length:length])
	return buffer
}

// MarshalMessage converts a message to native fields.
func MarshalMessage(msg Message) (keyLen C.size_t, keyPtr unsafe.Pointer, payLen C.size_t, payPtr unsafe.Pointer, usrLen C.size_t, usrPtr unsafe.Pointer) {
	key := msg.GetKey()
	pay := msg.GetPayload()
	usr := msg.GetUserdata()
	keyLen = C.size_t(len(key))
	payLen = C.size_t(len(pay))
	usrLen = C.size_t(len(usr))

	if keyLen > 0 {
		keyPtr = unsafe.Pointer(&key[0])
	} else {
		keyPtr = unsafe.Pointer(nil)
	}

	if payLen > 0 {
		payPtr = unsafe.Pointer(&pay[0])
	} else {
		payPtr = unsafe.Pointer(nil)
	}

	if usrLen > 0 {
		usrPtr = unsafe.Pointer(&usr[0])
	} else {
		usrPtr = unsafe.Pointer(nil)
	}

	return keyLen, keyPtr, payLen, payPtr, usrLen, usrPtr
}

// PrepareBatch converts a message array to an array of native messages.
// The resulting pointer has to be freed with C.free(unsafe.Pointer(p)).
func PrepareBatch(messages []Message) *C.rd_kafka_message_t {
	numMessages := C.int(len(messages))
	batch := C.CreateBatch(numMessages)

	for i := C.int(0); i < numMessages; i++ {
		keyLen, keyPtr, payLen, payPtr, usrLen, usrPtr := MarshalMessage(messages[i])
		C.StoreBatchItem(batch, i, keyLen, keyPtr, payLen, payPtr, usrLen, usrPtr)
	}

	return batch
}
