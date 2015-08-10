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
//     Address: "localhost:1463"
//     ConnectionBufferSizeKB: 1024
//     BatchMaxCount: 8192
//     BatchFlushCount: 4096
//     BatchTimeoutSec: 5
//     Filter: "filter.All"
//     Category:
//       "console" : "console"
//       "_GOLLUM_"  : "_GOLLUM_"
//     Stream:
//       - "console"
//       - "_GOLLUM_"
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
// Filter defines a filter function that removes or allows certain messages to
// pass through to scribe. By default this is set to filter.All.
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
	Filter          core.Filter
	batch           core.MessageBatch
	batchTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
	bufferSizeByte  int
}

const (
	scribeMetricName     = "Scribe:Messages-"
	scribeMetricRetry    = "Scribe:Retries"
	scribeMaxRetries     = 10
	scribeMaxSleepTimeMs = 3000
)

func init() {
	shared.TypeRegistry.Register(Scribe{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Scribe) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	plugin, err := core.NewPluginWithType(conf.GetString("Filter", "filter.All"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}

	prod.Filter = plugin.(core.Filter)
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

	shared.Metric.New(scribeMetricRetry)
	for _, category := range prod.category {
		shared.Metric.New(scribeMetricName + category)
	}
	return nil
}

func (prod *Scribe) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *Scribe) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
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

func (prod *Scribe) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Scribe) transformMessages(messages []core.Message) {
	logBuffer := make([]*scribe.LogEntry, len(messages))

	for idx, msg := range messages {
		msg.Data, msg.StreamID = prod.Format(msg)
		if !prod.Filter.Accepts(msg) {
			continue // ### continue, filtered ###
		}

		category, exists := prod.category[msg.StreamID]
		if !exists {
			if category, exists = prod.category[core.WildcardStreamID]; !exists {
				category = core.StreamRegistry.GetStreamName(msg.StreamID)
			}
			shared.Metric.New(scribeMetricName + category)
		}

		logBuffer[idx] = &scribe.LogEntry{
			Category: category,
			Message:  string(msg.Data),
		}

		shared.Metric.Inc(scribeMetricName + category)
	}

	// Retry messages
	for retryCount := 0; retryCount < scribeMaxRetries; retryCount++ {
		resultCode, err := prod.scribe.Log(logBuffer)
		if resultCode == scribe.ResultCode_OK {
			return // ### return, success ###
		}

		if err != nil || resultCode != scribe.ResultCode_TRY_LATER {
			Log.Error.Printf("Scribe log error %d: %s", resultCode, err.Error())
			prod.transport.Close() // reconnect
			prod.dropMessages(messages)
			return // ### return, failure ###
		}

		shared.Metric.Inc(scribeMetricRetry)
		time.Sleep(time.Duration(scribeMaxSleepTimeMs/scribeMaxRetries) * time.Millisecond)
	}

	Log.Error.Printf("Scribe server seems to be busy")
	prod.dropMessages(messages)
}

func (prod *Scribe) close() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.WorkerDone()
	}()

	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
