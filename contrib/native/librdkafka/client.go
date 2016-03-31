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

// Client is a wrapper handle for rd_kafka_t
type Client struct {
	handle *C.rd_kafka_t
}

var (
	clients = make(map[*C.rd_kafka_t]MessageDelivery)
)

// NewProducer creates a new librdkafka client in producer mode.
// Make sure to call Close() when done.
func NewProducer(config Config, handler MessageDelivery) (*Client, error) {
	client := Client{}
	nativeErr := new(ErrorHandle)

	C.RegisterErrorWrapper(config.handle)
	C.RegisterDeliveryReportWrapper(config.handle)

	client.handle = C.rd_kafka_new(C.RD_KAFKA_PRODUCER, config.handle, nativeErr.buffer(), nativeErr.len())
	if client.handle == nil {
		return nil, nativeErr
	}

	if handler != nil {
		clients[client.handle] = handler
	}

	return &client, nil
}

// Close frees the native handle.
func (t *Client) Close() {
	C.rd_kafka_destroy(t.handle)
}

// GetInflightBuffers returns the number of allocated buffers (message useradata)
func (t *Client) GetInflightBuffers() int64 {
	return int64(C.GetAllocatedBuffers())
}
