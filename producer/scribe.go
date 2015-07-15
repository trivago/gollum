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
//     BatchMaxCount: 16384
//     BatchFlushCount: 10000
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
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
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
	scribe          *scribe.ScribeClient
	transport       *thrift.TFramedTransport
	socket          *thrift.TSocket
	category        map[core.MessageStreamID]string
	batch           core.MessageBatch
	batchTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
	bufferSizeByte  int
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

	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("BatchFlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = shared.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.batch = core.NewMessageBatch(prod.batchMaxCount)

	prod.bufferSizeByte = conf.GetInt("ConnectionBufferSizeKB", 1<<10) << 10 // 1 MB
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
			prod.socket.Conn().(bufferedConn).SetWriteBuffer(prod.bufferSizeByte)
		}
	}

	if prod.transport.IsOpen() {
		prod.batch.Flush(prod.transformMessages)
	}
}

func (prod *Scribe) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *Scribe) sendMessage(msg core.Message) {
	if !prod.batch.Append(msg) {
		prod.sendBatch()
		if !prod.batch.AppendOrBlock(msg) {
			prod.Drop(msg)
		}
	}
}

func (prod *Scribe) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Scribe) transformMessages(messages []core.Message) {
	logBuffer := make([]*scribe.LogEntry, len(messages))

	for idx, msg := range messages {
		data, streamID := prod.Format(msg)
		category, exists := prod.category[streamID]
		if !exists {
			if category, exists = prod.category[core.WildcardStreamID]; !exists {
				category = core.StreamTypes.GetStreamName(streamID)
			}
		}

		logBuffer[idx] = &scribe.LogEntry{
			Category: category,
			Message:  string(data),
		}
	}

	_, err := prod.scribe.Log(logBuffer)
	if err != nil {
		Log.Error.Print("Scribe log error: ", err)
		prod.transport.Close()
		prod.dropMessages(messages)
	}
}

// Close gracefully
func (prod *Scribe) Close() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.WorkerDone()
	}()

	if prod.CloseGracefully(prod.sendMessage) {
		prod.batch.Close()
		prod.sendBatch()
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}

	if !prod.batch.IsEmpty() {
		prod.batch.Close()
		prod.batch.Flush(prod.dropMessages)
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.batchTimeout, prod.sendMessage, nil, prod.sendBatchOnTimeOut)
}
