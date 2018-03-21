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

// +build cgo,!unit

package librdkafka

// #cgo CFLAGS: -I/usr/local/include -std=c99 -Wno-deprecated-declarations
// #cgo LDFLAGS: -L/usr/local/lib -L/usr/local/opt/librdkafka/lib -lrdkafka
// #include "wrapper.h"
import "C"

import (
	"reflect"
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
	OnMessageDelivered(userdata []byte)
}

//export goDeliveryHandler
func goDeliveryHandler(clientHandle *C.rd_kafka_t, code C.int, bufferPtr *C.buffer_t) {
	clientGuard.RLock()
	handler, handlerExists := clients[clientHandle]
	clientGuard.RUnlock()

	if handlerExists {
		buffer := UnmarshalBuffer(bufferPtr)
		if code == C.RD_KAFKA_RESP_ERR_NO_ERROR {
			handler.OnMessageDelivered(buffer)
		} else {
			handler.OnMessageError(codeToString(int(code)), buffer)
		}
	}
}

// UnmarshalBuffer creates a byte slice copy from a buffer_t handle.
func UnmarshalBuffer(bufferPtr *C.buffer_t) []byte {
	length := int(bufferPtr.len)
	buffer := make([]byte, length)
	copy(buffer, (*[1 << 30]byte)(bufferPtr.data)[:length:length])
	return buffer
}

type nativeMessage struct {
	key        unsafe.Pointer
	payload    unsafe.Pointer
	keyLen     C.size_t
	payloadLen C.size_t
}

func newBuffer(data []byte) *C.buffer_t {
	size := C.size_t(len(data))
	if size == 0 {
		return (*C.buffer_t)(nil)
	}

	nativeCopy := allocCopy(data)
	return C.CreateBuffer(size, nativeCopy)
}

func freeBuffer(buffer *C.buffer_t) {
	C.DestroyBuffer(buffer)
}

func newNativeMessage(msg Message) nativeMessage {
	key := msg.GetKey()
	pay := msg.GetPayload()

	return nativeMessage{
		key:        allocCopy(key),
		keyLen:     C.size_t(len(key)),
		payload:    allocCopy(pay),
		payloadLen: C.size_t(len(pay)),
	}
}

func allocCopy(data []byte) unsafe.Pointer {
	size := C.size_t(len(data))
	if size == 0 {
		return unsafe.Pointer(nil)
	}

	nativePtr := C.Alloc(size)
	slice := unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(nativePtr),
		Len:  len(data),
		Cap:  len(data),
	})

	byteSlice := (*[]byte)(slice)
	copy(*byteSlice, data)
	return nativePtr
}

func (n *nativeMessage) free() {
	C.Free(n.key)
	C.Free(n.payload)
}
