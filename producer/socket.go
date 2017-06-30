// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tnet"
	"net"
	"sync"
	"time"
)

// Socket producer plugin
//
// The socket producer connects to a service over a TCP, UDP or unix domain
// socket based connection.
//
// Configuration example
//
//  - "producer.Socket":
//    Enable: true
//    Address: ":5880"
//    ConnectionBufferSizeKB: 1024
//    BatchMaxCount: 8192
//    BatchFlushCount: 4096
//    BatchTimeoutSec: 5
//    Acknowledge: ""
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// ConnectionBufferSizeKB sets the connection buffer size in KB. By default this
// is set to 1024, i.e. 1 MB buffer.
//
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block. By default this is set to 8192.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
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
//
// AckTimeoutMs defines the time in milliseconds to wait for a response from the
// server. After this timeout the send is marked as failed. Defaults to 2000.
type Socket struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	connection            net.Conn
	batch                 core.MessageBatch
	assembly              core.WriterAssembly
	protocol              string
	address               string
	batchTimeout          time.Duration `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	ackTimeout            time.Duration `config:"AckTimeoutMs" default:"2000" metric:"ms"`
	batchMaxCount         int           `config:"Batch/MaxCount" default:"8192"`
	batchFlushCount       int           `config:"Batch/FlushCount" default:"4096"`
	bufferSizeByte        int           `config:"ConnectionBufferSizeKB" default:"1024" metric:"kb"`
	acknowledge           string        `config:"Acknowledge"`
}

type bufferedConn interface {
	SetWriteBuffer(bytes int) error
}

func init() {
	core.TypeRegistry.Register(Socket{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Socket) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	prod.protocol, prod.address = tnet.ParseAddress(conf.GetString("Address", ":5880"), "tcp")
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)

	switch prod.protocol {
	case "udp":
		if prod.acknowledge != "" {
			prod.Logger.Warning("Acknowledge is only supported for TCP connections. TCP connection forced.")
			prod.protocol = "tcp"
		}
	case "unix", "tcp":
		// Everything is fine
	default:
		prod.protocol = "tcp"
	}

	prod.batch = core.NewMessageBatch(prod.batchMaxCount)
	prod.assembly = core.NewWriterAssembly(nil, prod.TryFallback, prod)
	prod.assembly.SetValidator(prod.validate)
	prod.assembly.SetErrorHandler(prod.onWriteError)
}

func (prod *Socket) tryConnect() bool {
	if prod.connection != nil {
		return true // ### return, connection active ###
	}

	conn, err := net.DialTimeout(prod.protocol, prod.address, prod.ackTimeout)
	if err != nil {
		prod.Logger.Error("Connection error: ", err)
		prod.closeConnection()
		return false // ### return, connection failed ###
	}

	conn.(bufferedConn).SetWriteBuffer(prod.bufferSizeByte)
	prod.assembly.SetWriter(conn)
	prod.connection = conn
	return true
}

func (prod *Socket) closeConnection() error {
	prod.assembly.SetWriter(nil)
	if prod.connection != nil {
		prod.connection.Close()
		prod.connection = nil

		if !prod.IsStopping() {
			// TBD: action needed? (ex-fuse breaker)
		}
	}
	return nil
}

func (prod *Socket) validate() bool {
	if prod.acknowledge == "" {
		return true
	}

	response := make([]byte, len(prod.acknowledge))
	prod.connection.SetReadDeadline(time.Now().Add(prod.ackTimeout))
	_, err := prod.connection.Read(response)
	if err != nil {
		prod.Logger.Error("Response error: ", err)
		if tnet.IsDisconnectedError(err) {
			prod.closeConnection()
		}
		return false
	}
	return string(response) == prod.acknowledge
}

func (prod *Socket) onWriteError(err error) bool {
	prod.Logger.Error("Write error: ", err)
	prod.closeConnection()
	return false
}

func (prod *Socket) sendMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *Socket) sendBatch() {
	// Flush the buffer to the connection if it is active
	if prod.tryConnect() {
		prod.batch.Flush(prod.assembly.Write)
	} else if prod.IsStopping() {
		prod.batch.Flush(prod.assembly.Flush)
	}
}

func (prod *Socket) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *Socket) close() {
	defer func() {
		prod.batch.AfterFlushDo(prod.closeConnection)
		prod.WorkerDone()
	}()

	prod.DefaultClose()

	if prod.tryConnect() {
		prod.batch.Close(prod.assembly.Write, prod.GetShutdownTimeout())
	} else {
		prod.batch.Close(prod.assembly.Flush, prod.GetShutdownTimeout())
	}
}

// Produce writes to a buffer that is sent to a given socket.
func (prod *Socket) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.sendMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
