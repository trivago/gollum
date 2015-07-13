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
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"sync"
	"time"
)

// Scribe producer plugin
// Configuration example
//
//   - "producer.Scribe":
//     Enable: true
//     Address: "192.168.222.30:1463"
//     ConnectionBufferSizeKB: 4096
//     BatchSizeMaxKB: 16384
//     BatchSizeByte: 4096
//     BatchTimeoutSec: 2
//     Stream:
//       - "console"
//       - "_GOLLUM_"
//     Category:
//       "console" : "default"
//       "_GOLLUM_"  : "default"
//
// The scribe producer allows sending messages to Facebook's scribe.
//
// Address defines the host and port to connect to.
// By default this is set to "localhost:1463".
//
// ConnectionBufferSizeKB sets the connection buffer size in KB. By default this
// is set to 1024, i.e. 1 MB buffer.
//
// BatchSizeMaxKB defines the maximum number of bytes to buffer before
// messages get dropped. If a message crosses the threshold it is still buffered
// but additional messages will be dropped. By default this is set to 8192.
//
// BatchSizeByte defines the number of bytes to be buffered before they are written
// to scribe. By default this is set to 8KB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// Category maps a stream to a specific scribe category. You can define the
// wildcard stream (*) here, too. When set, all streams that do not have a
// specific mapping will go to this category (including _GOLLUM_).
// If no category mappings are set the stream name is used.
type Scribe struct {
	core.ProducerBase
	scribe       *scribe.ScribeClient
	transport    *thrift.TFramedTransport
	socket       *thrift.TSocket
	batch        *scribeMessageBatch
	category     map[core.MessageStreamID]string
	batchSize    int
	batchTimeout time.Duration
	bufferSizeKB int
}

func init() {
	shared.RuntimeType.Register(Scribe{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Scribe) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	host := conf.GetString("Address", "localhost:1463")
	bufferSizeMax := conf.GetInt("BatchSizeMaxKB", 8<<10) << 1 // 8 MB

	prod.category = make(map[core.MessageStreamID]string, 0)
	prod.batchSize = conf.GetInt("BatchSizeByte", 8192)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.batch = createScribeMessageBatch(bufferSizeMax)
	prod.bufferSizeKB = conf.GetInt("ConnectionBufferSizeKB", 1<<10) // 1 MB
	prod.category = conf.GetStreamMap("Category", "")

	// Initialize scribe connection

	prod.socket, err = thrift.NewTSocket(host)
	if err != nil {
		Log.Error.Print("Scribe socket error:", err)
		return err
	}

	prod.transport = thrift.NewTFramedTransport(prod.socket)
	binProtocol := thrift.NewTBinaryProtocol(prod.transport, false, false)
	prod.scribe = scribe.NewScribeClientProtocol(prod.transport, binProtocol, binProtocol)

	return nil
}

func (prod *Scribe) sendBatch() {
	if !prod.transport.IsOpen() {
		err := prod.transport.Open()
		if err != nil {
			Log.Error.Print("Scribe connection error:", err)
		} else {
			prod.socket.Conn().(bufferedConn).SetWriteBuffer(prod.bufferSizeKB << 10)
		}
	}

	if prod.transport.IsOpen() {
		prod.batch.flush(prod.scribe, func(err error) {
			Log.Error.Print("Scribe log error: ", err)
			prod.transport.Close()
		})
	}
}

func (prod *Scribe) sendBatchOnTimeOut() {
	if prod.batch.reachedTimeThreshold(prod.batchTimeout) || prod.batch.reachedSizeThreshold(prod.batchSize) {
		prod.sendBatch()
	}
}

func (prod *Scribe) sendMessage(msg core.Message) {
	msg.Data, msg.StreamID = prod.ProducerBase.Format(msg)

	category, exists := prod.category[msg.StreamID]
	if !exists {
		category, exists = prod.category[core.WildcardStreamID]
		if !exists {
			category = core.StreamTypes.GetStreamName(msg.StreamID)
		}
	}

	if !prod.batch.Append(msg, category) {
		prod.sendBatch()
		prod.batch.AppendOrBlock(msg, category)
	}
}

// Close gracefully
func (prod *Scribe) Close() {
	defer prod.WorkerDone()
	if prod.CloseGracefully(prod.sendMessage) {
		prod.sendBatch()
		prod.batch.waitForFlush(prod.GetShutdownTimeout())

		prod.transport.Close()
		prod.socket.Close()
	}

	if !prod.batch.isEmpty() {
		prod.batch.drop(prod)
	}
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.batchTimeout, prod.sendMessage, nil, prod.sendBatchOnTimeOut)
}
