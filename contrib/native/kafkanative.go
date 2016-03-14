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

package native

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/opt/librdkafka/lib -L/usr/local/lib -lrdkafka
// #include <librdkafka/rdkafka.h>
import "C"

import (
	"strconv"
	"unsafe"
)

type libKafkaError struct {
	errBuffer [512]byte
}

func (l *libKafkaError) buffer() *C.char {
	return (*C.char)(unsafe.Pointer(&l.errBuffer[0]))
}

func (l *libKafkaError) len() C.size_t {
	return C.size_t(len(l.errBuffer))
}

func (l *libKafkaError) Error() string {
	for i := 0; i < len(l.errBuffer); i++ {
		if l.errBuffer[i] == 0 {
			return string(l.errBuffer[:i])
		}
	}

	return string(l.errBuffer[:len(l.errBuffer)])
}

type libKafkaConfig struct {
	handle *C.struct_rd_kafka_conf_s
}

func newLibKafkaConfig() libKafkaConfig {
	return libKafkaConfig{
		handle: C.rd_kafka_conf_new(),
	}
}

func (c *libKafkaConfig) Set(key, value string) error {
	nativeErr := new(libKafkaError)
	if C.rd_kafka_conf_set(c.handle, C.CString(key), C.CString(value), nativeErr.buffer(), nativeErr.len()) != 0 {
		return nativeErr
	}
	return nil
}

func (c *libKafkaConfig) SetI(key string, value int) error {
	nativeErr := new(libKafkaError)
	strValue := strconv.Itoa(value)
	if C.rd_kafka_conf_set(c.handle, C.CString(key), C.CString(strValue), nativeErr.buffer(), nativeErr.len()) != 0 {
		return nativeErr
	}
	return nil
}

func (c *libKafkaConfig) SetB(key string, value bool) error {
	nativeErr := new(libKafkaError)
	var boolValue string
	if value {
		boolValue = "true"
	} else {
		boolValue = "false"
	}
	if C.rd_kafka_conf_set(c.handle, C.CString(key), C.CString(boolValue), nativeErr.buffer(), nativeErr.len()) != 0 {
		return nativeErr
	}
	return nil
}

type libKafkaProducer struct {
	handle *C.struct_rd_kafka_s
}

func newLibKafkaProducer(config libKafkaConfig) (libKafkaProducer, error) {
	nativeErr := new(libKafkaError)
	client := C.rd_kafka_new(C.RD_KAFKA_PRODUCER, config.handle, nativeErr.buffer(), nativeErr.len())
	if client == nil {
		return libKafkaProducer{}, nativeErr
	}
	return libKafkaProducer{client}, nil
}
