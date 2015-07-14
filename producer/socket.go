// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"net"
	"sync"
	"time"
)

// Socket producer plugin
// Configuration example
//
//   - "producer.Socket":
//     Enable: true
//     Address: "unix:///var/gollum.socket"
//     ConnectionBufferSizeKB: 4096
//     BatchSizeMaxKB: 16384
//     BatchSizeByte: 4096
//     BatchTimeoutSec: 5
//     Acknowledge: "ACK\n"
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// ConnectionBufferSizeKB sets the connection buffer size in KB. By default this
// is set to 1024, i.e. 1 MB buffer.
//
// BatchSizeMaxKB defines the maximum number of bytes to buffer before
// messages get dropped. Any message that crosses the threshold is dropped.
// By default this is set to 8192.
//
// BatchSizeByte defines the number of bytes to be buffered before they are written
// to scribe. By default this is set to 8KB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// Acknowledge can be set to a non-empty value to expect the given string as a
// response from the server after a batch has been sent.
// This setting is disabled by default, i.e. set to "".
// If Acknowledge is enabled and a IP-Address is given to Address, TCP is used
// to open the connection, otherwise UDP is used.
type Socket struct {
	core.ProducerBase
	connection      net.Conn
	batch           core.MessageBatch
	assembly        core.WriterAssembly
	protocol        string
	address         string
	batchTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
	bufferSizeKB    int
	acknowledge     string
}

type bufferedConn interface {
	SetWriteBuffer(bytes int) error
}

func init() {
	shared.RuntimeType.Register(Socket{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Socket) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("BatchFlushCount", prod.batchMaxCount/2)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.bufferSizeKB = conf.GetInt("ConnectionBufferSizeKB", 1<<10) // 1 MB

	prod.acknowledge = shared.Unescape(conf.GetString("Acknowledge", ""))
	prod.address, prod.protocol = shared.ParseAddress(conf.GetString("Address", ":5880"))

	if prod.protocol != "unix" {
		if prod.acknowledge != "" {
			prod.protocol = "tcp"
		} else {
			prod.protocol = "udp"
		}
	}

	prod.batch = core.NewMessageBatch(prod.batchMaxCount)
	prod.assembly = core.NewWriterAssembly(prod.connection, prod.GetFormatter(), prod.GetDropStreamID())
	return nil
}

func (prod *Socket) validate() bool {
	if prod.acknowledge == "" {
		return true
	}

	response := make([]byte, len(prod.acknowledge))
	_, err := prod.connection.Read(response)
	if err != nil {
		Log.Error.Print("Socket response error:", err)
		return false
	}
	return string(response) == prod.acknowledge
}

func (prod *Socket) onWriteError(err error) bool {
	Log.Error.Print("Socket error - ", err)
	prod.connection.Close()
	prod.connection = nil
	return false
}

func (prod *Socket) sendBatch() {
	// If we have not yet connected or the connection dropped: connect.
	if prod.connection == nil {
		conn, err := net.Dial(prod.protocol, prod.address)

		if err != nil {
			Log.Error.Print("Socket connection error - ", err)
		} else {
			conn.(bufferedConn).SetWriteBuffer(prod.bufferSizeKB << 10)
			prod.connection = conn
		}
	}

	// Flush the buffer to the connection if it is active
	if prod.connection != nil {
		prod.batch.Flush(prod.assembly.Write)
	}
}

func (prod *Socket) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *Socket) sendMessage(message core.Message) {
	if !prod.batch.Append(message) {
		prod.sendBatch()
		if !prod.batch.AppendOrBlock(message) {
			prod.Drop(message)
		}
	}
}

// Close gracefully
func (prod *Socket) Close() {
	defer func() {
		if prod.connection != nil {
			prod.connection.Close()
		}
		prod.WorkerDone()
	}()

	// Flush buffer to regular socket
	if prod.CloseGracefully(prod.sendMessage) {
		prod.batch.Close()
		prod.sendBatch()
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}

	// Drop all data that is still in the buffer
	if !prod.batch.IsEmpty() {
		prod.batch.Close()
		prod.batch.Flush(prod.assembly.Drop)
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *Socket) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.batchTimeout, prod.sendMessage, nil, prod.sendBatchOnTimeOut)
}
