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
	"fmt"
	"unsafe"
)

var (
	// required for getting from C back to Go as we cannot pass Go structs with
	// Go pointers to C.
	allTopics = []*Topic{}
)

// Topic wrapper handle for rd_kafka_topic_t
type Topic struct {
	handle      *C.rd_kafka_topic_t
	client      *Client
	name        string
	transmitErr []asyncError
	id          int
}

type asyncError struct {
	code  int
	index int
}

// NewTopic creates a new topic representation in librdkafka.
// You have to call Close() to free any internal state. As this struct holds a
// pointer to the client make sure that Client.Close is called after closing
// objects of this type.
func NewTopic(name string, p Partitioner, config TopicConfig, client *Client) (*Topic, error) {
	topic := &Topic{
		handle:      C.rd_kafka_topic_new(client.handle, C.CString(name), config.handle),
		client:      client,
		name:        name,
		transmitErr: []asyncError{},
		id:          len(allTopics),
	}

	C.RegisterRandomPartitioner(topic.handle)

	// TODO: Locking?
	allTopics = append(allTopics, topic)
	return topic, nil
}

// Close frees the internal handle
func (t *Topic) Close() {
	C.rd_kafka_topic_destroy(t.handle)
}

// GetName returns the name of the topic
func (t *Topic) GetName() string {
	return t.name
}

// Produce sends the list of messages given to kafka and blocks until all
// messages have been sent.
func (t *Topic) Produce(messages []Message) []ResponseError {
	errors := []ResponseError{}
	if len(messages) == 0 {
		return errors // ### return, nothing to do ###
	}

	batch := PrepareBatch(messages, t)
	batchLen := C.int(len(messages))
	defer C.DestroyBatch(unsafe.Pointer(batch), C.int(len(messages)))
	t.transmitErr = t.transmitErr[:0] // Clear

	if enqueued := C.rd_kafka_produce_batch(t.handle, C.RD_KAFKA_PARTITION_UA, C.RD_KAFKA_MSG_F_COPY, batch, batchLen); enqueued != batchLen {
		fmt.Println("enqueued", enqueued, "of", batchLen)

		offset := C.int(0)
		for offset >= 0 {
			offset = C.NextError(batch, batchLen, offset)
			if offset >= 0 {
				rspErr := ResponseError{
					Original: messages[offset],
					Code:     int(C.GetErr(batch, offset)),
				}
				errors = append(errors, rspErr)
				offset++
			}
		}
	}

	fmt.Println("polling")

	for i := 0; C.rd_kafka_outq_len(t.client.handle) > 0 && i < 100; i++ {
		C.rd_kafka_poll(t.client.handle, 10)
	}

	if C.rd_kafka_outq_len(t.client.handle) > 0 {
		fmt.Println(C.rd_kafka_outq_len(t.client.handle), "queued")
	} else {
		fmt.Println("done")
	}

	for _, err := range t.transmitErr {
		rspErr := ResponseError{
			Original: messages[err.index],
			Code:     err.code,
		}
		errors = append(errors, rspErr)
	}

	return errors
}

func (t *Topic) pushError(code int, index int) {
	err := asyncError{
		code:  code,
		index: index,
	}
	t.transmitErr = append(t.transmitErr, err)
}
