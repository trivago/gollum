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

package producer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tnet"
	"github.com/trivago/tgo/tstrings"
	"net"
	"strings"
	"sync"
	"time"
)

// Proxy producer plugin
//
// This producer is a compatible with the Proxy consumer plugin.
// Responses to messages sent to the given address are sent back to the original
// consumer of it is a compatible message source. As with consumer.proxy the
// returned messages are partitioned by common message length algorithms.
//
// Parameters
//
// - Address: This value stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.Proxy".
// By default this parameter is set to ":5880".
//
// - ConnectionBufferSizeKB: This value sets the connection buffer size in KB.
// This also defines the size of the buffer used by the message parser.
// By default this parameter is set to "1024".
//
// - TimeoutSec: This value defines the maximum time in seconds a client is allowed to take
// for a response.
// By default this parameter is set to "1".
//
// - Partitioner: This value defines the algorithm used to read messages from the stream.
// The messages will be sent as a whole, no cropping or removal will take place.
// By default this parameter is set to "delimiter".
//
//  - delimiter: separates messages by looking for a delimiter string. The
//  delimiter is included into the left hand message.
//
//  - ascii: reads an ASCII encoded number at a given offset until a given
//  delimiter is found.
//
//  - binary: reads a binary number at a given offset and size
//
//  - binary_le: is an alias for "binary"
//
//  - binary_be: is the same as "binary" but uses big endian encoding
//
//  - fixed: assumes fixed size messages
//
// - Delimiter: This value defines the delimiter used by the text and delimiter partitioner.
// By default this parameter is set to "\n".
//
// - Offset: This value defines the offset used by the binary and text partitioner.
// This setting is ignored by the fixed partitioner.
// By default this parameter is set to "0".
//
// - Size: This value defines the size in bytes used by the binary or fixed partitioner.
// For `binary` this can be set to 1,2,4 or 8,  for `fixed` this defines the size of a message.
// BY default this parameter is set to "4" for `binary` or "1" for `fixed` partitioner.
//
// Examples
//
// This example will send 64bit length encoded data on TCP port 5880.
//
//  proxyOut:
//    Type: producer.Proxy
//    Address: ":5880"
//    Partitioner: binary
//    Size: 8
//
type Proxy struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	connection            net.Conn
	protocol              string
	address               string
	bufferSizeKB          int           `config:"ConnectionBufferSizeKB" default:"1024" metric:"mb"`
	timeout               time.Duration `config:"TimeoutSec" default:"1" metric:"sec"`
	reader                *tio.BufferedReader
}

func init() {
	core.TypeRegistry.Register(Proxy{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Proxy) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	prod.protocol, prod.address = tnet.ParseAddress(conf.GetString("Address", ":5880"), "tcp")
	if prod.protocol == "udp" {
		conf.Errors.Pushf("Proxy does not support UDP")
	}

	delimiter := tstrings.Unescape(conf.GetString("Delimiter", "\n"))
	offset := int(conf.GetInt("Offset", 0))
	flags := tio.BufferedReaderFlagEverything // pass all messages as-is

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		flags |= tio.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
		switch conf.GetInt("Size", 4) {
		case 1:
			flags |= tio.BufferedReaderFlagMLE8
		case 2:
			flags |= tio.BufferedReaderFlagMLE16
		case 4:
			flags |= tio.BufferedReaderFlagMLE32
		case 8:
			flags |= tio.BufferedReaderFlagMLE64
		default:
			conf.Errors.Pushf("Size only supports the value 1,2,4 and 8")
		}

	case "fixed":
		flags |= tio.BufferedReaderFlagMLEFixed
		offset = int(conf.GetInt("Size", 1))

	case "ascii":
		flags |= tio.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		conf.Errors.Pushf("Unknown partitioner: %s", partitioner)
	}

	prod.reader = tio.NewBufferedReader(prod.bufferSizeKB, flags, offset, delimiter)
}

func (prod *Proxy) sendMessage(msg *core.Message) {
	// If we have not yet connected or the connection sent to the fallback: connect.
	for prod.connection == nil {
		conn, err := net.DialTimeout(prod.protocol, prod.address, prod.timeout)

		if err != nil {
			prod.Logger.Error("Connection error - ", err)
			<-time.After(time.Second)
		} else {
			conn.(bufferedConn).SetWriteBuffer(prod.bufferSizeKB << 10)
			prod.connection = conn
		}
	}

	// Check if and how to work with the message source
	responder, processResponse := msg.GetSource().(core.AsyncMessageSource)
	if processResponse {
		if serialResponder, isSerial := msg.GetSource().(core.SerialMessageSource); isSerial {
			defer serialResponder.ResponseDone()
		}
	}

	// Write data
	prod.connection.SetWriteDeadline(time.Now().Add(prod.timeout))
	if _, err := prod.connection.Write(msg.GetPayload()); err != nil {
		prod.Logger.Error("Write error: ", err)
		prod.connection.Close()
		prod.connection = nil
		return // ### return, connection closed ###
	}

	// Prepare responder function
	enqueueResponse := tio.BufferReadCallback(nil)
	if processResponse {
		enqueueResponse = func(data []byte) {
			response := core.NewMessage(prod, data, nil, msg.GetStreamID())
			responder.EnqueueResponse(response)
		}
	}

	// Read response
	prod.connection.SetReadDeadline(time.Now().Add(prod.timeout))
	if err := prod.reader.ReadAll(prod.connection, enqueueResponse); err != nil {
		prod.Logger.Error("Read error: ", err)
		prod.connection.Close()
		prod.connection = nil
		return // ### return, connection closed ###
	}
}

func (prod *Proxy) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()

	if prod.connection != nil {
		prod.connection.Close()
	}
}

// Produce writes to a buffer that is sent to a given Proxy.
func (prod *Proxy) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.sendMessage)
}
