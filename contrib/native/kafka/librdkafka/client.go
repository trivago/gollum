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
	"sync"
	"time"
)

// Client is a wrapper handle for rd_kafka_t
type Client struct {
	handle *C.rd_kafka_t
}

var (
	clientGuard = new(sync.RWMutex)
	clients     = make(map[*C.rd_kafka_t]MessageDelivery)
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
		clientGuard.Lock()
		clients[client.handle] = handler
		clientGuard.Unlock()
	}

	return &client, nil
}

// Poll polls for new data to be sent to the async handler functions
func (cl *Client) Poll(timeout time.Duration) {
	if timeout < 0 {
		C.rd_kafka_poll(cl.handle, -1)
	} else {
		timeoutMs := C.int(timeout.Nanoseconds() / 1000000)
		C.rd_kafka_poll(cl.handle, timeoutMs)
	}
}

// Close frees the native handle.
func (cl *Client) Close() {
	C.rd_kafka_destroy(cl.handle)
}

// GetAllocCounter returns the number of allocated native buffers
func (cl *Client) GetAllocCounter() int64 {
	return int64(C.GetAllocCounter())
}
