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

// Topic wrapper handle for rd_kafka_topic_t
type Topic struct {
	handle   *C.rd_kafka_topic_t
	client   *Client
	name     string
	shutdown bool
}

// NewTopic creates a new topic representation in librdkafka.
// You have to call Close() to free any internal state. As this struct holds a
// pointer to the client make sure that Client.Close is called after closing
// objects of this type.
func NewTopic(name string, config TopicConfig, client *Client) *Topic {
	return &Topic{
		handle:   C.rd_kafka_topic_new(client.handle, C.CString(name), config.handle),
		client:   client,
		name:     name,
		shutdown: false,
	}
}

// TriggerShutdown signals a topic to stop producing messages (unblocks any
// waiting topics).
func (t *Topic) TriggerShutdown() {
	t.shutdown = true
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
		Log.Printf("%d messages have been lost as the internal queue could not be flushed", queueLen)
	}
	C.rd_kafka_topic_destroy(t.handle)
}

// GetName returns the name of the topic
func (t *Topic) GetName() string {
	return t.name
}

// Poll polls for new data to be sent to the async handler functions
func (t *Topic) Poll() {
	C.rd_kafka_poll(t.client.handle, 1000)
}

// Produce produces a single messages.
// If a message cannot be produced because of internal (non-wire) problems an
// error is immediately returned instead of being asynchronously handled via
// MessageDelivery interface.
func (t *Topic) Produce(message Message) error {
	keyLen, keyPtr, payLen, payPtr, usrLen, usrPtr := MarshalMessage(message)
	usrData := C.CreateBuffer(usrLen, usrPtr)
	success := C.rd_kafka_produce(t.handle, C.RD_KAFKA_PARTITION_UA, C.RD_KAFKA_MSG_F_COPY, payPtr, payLen, keyPtr, keyLen, usrData)

	if success != 0 {
		defer C.DestroyBuffer(usrData)
		rspErr := ResponseError{
			Userdata: message.GetUserdata(),
			Code:     int(C.GetLastError()),
		}
		return rspErr // ### return, error ###
	}

	C.rd_kafka_poll(t.client.handle, 0)
	return nil
}

// ProduceBatch produces a set of messages.
// Messages that cannot be produced because of internal (non-wire) problems are
// immediately returned instead of asynchronously handled via MessageDelivery
// interface.
/*func (t *Topic) ProduceBatch(messages []Message) []error {
	errors := []error{}
	if len(messages) == 0 {
		return errors // ### return, nothing to do ###
	}

	batch := PrepareBatch(messages)
	batchLen := C.int(len(messages))
	defer C.DestroyBatch(unsafe.Pointer(batch))

	enqueued := C.rd_kafka_produce_batch(t.handle, C.RD_KAFKA_PARTITION_UA, C.RD_KAFKA_MSG_F_COPY, batch, batchLen)
	if enqueued != batchLen {
		offset := C.int(0)
		for offset >= 0 {
			offset = C.BatchGetNextError(batch, batchLen, offset)
			if offset >= 0 {
				bufferPtr := C.BatchGetUserdataAt(batch, offset)
				errCode := C.BatchGetErrAt(batch, offset)

				rspErr := ResponseError{
					Userdata: UnmarshalBuffer(bufferPtr),
					Code:     int(errCode),
				}

				errors = append(errors, rspErr)
				offset++
			}
		}
	}

	for C.rd_kafka_outq_len(t.client.handle) > 0 && !t.shutdown {
		C.rd_kafka_poll(t.client.handle, 20)
	}

	return errors
}*/
