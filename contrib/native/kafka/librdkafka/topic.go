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

// Topic wrapper handle for rd_kafka_topic_t
type Topic struct {
	handle *C.rd_kafka_topic_t
	client *Client
	name   string
}

// NewTopic creates a new topic representation in librdkafka.
// You have to call Close() to free any internal state. As this struct holds a
// pointer to the client make sure that Client.Close is called after closing
// objects of this type.
func NewTopic(name string, config TopicConfig, client *Client) *Topic {
	return &Topic{
		handle: C.rd_kafka_topic_new(client.handle, C.CString(name), config.handle),
		client: client,
		name:   name,
	}
}

// Close frees the internal handle and tries to flush the queue.
func (t *Topic) Close() {
	oldQueueLen := C.int(0x7FFFFFFF)
	queueLen := C.rd_kafka_outq_len(t.client.handle)

	// Wait as long as we're flushing
	for queueLen > 0 && queueLen < oldQueueLen {
		C.rd_kafka_poll(t.client.handle, 1000)
		oldQueueLen = queueLen
		queueLen = C.rd_kafka_outq_len(t.client.handle)
	}

	if queueLen > 0 {
		Log.Printf("%d messages have been lost as the internal queue for %s could not be flushed", queueLen, t.name)
	}
	C.rd_kafka_topic_destroy(t.handle)
}

// GetName returns the name of the topic
func (t *Topic) GetName() string {
	return t.name
}

// Produce produces a single messages.
// If a message cannot be produced because of internal (non-wire) problems an
// error is immediately returned instead of being asynchronously handled via
// MessageDelivery interface.
func (t *Topic) Produce(msg Message) error {
	userdata := newBuffer(msg.GetUserdata())
	native := newNativeMessage(msg)
	defer native.free()

	if C.Produce(t.handle, native.key, native.keyLen, native.payload, native.payloadLen, userdata) != 0 {
		defer freeBuffer(userdata)
		rspErr := ResponseError{
			Userdata: msg.GetUserdata(),
			Code:     int(C.GetLastError()),
		}
		if rspErr.Code == C.RD_KAFKA_RESP_ERR__QUEUE_FULL {
			t.client.Poll(-1)
		}
		return rspErr // ### return, error ###
	}
	return nil
}
