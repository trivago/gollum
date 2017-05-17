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
	"github.com/artyom/fb303"
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tmath"
	"sync"
	"time"
)

// Scribe producer plugin
///
// The scribe producer allows sending messages to Facebook's scribe.
//
// Configuration example
//
//  - "producer.Scribe":
//    Address: "localhost:1463"
//    ConnectionBufferSizeKB: 1024
//    BatchMaxCount: 8192
//    BatchFlushCount: 4096
//    BatchTimeoutSec: 5
//	  HeartBeatIntervalSec: 5
//    Category:
//      "console" : "console"
//      "_GOLLUM_"  : "_GOLLUM_"
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
// set to 5. This also defines the maximum time allowed for messages to be
// sent to the server.
//
// HeartBeatIntervalSec defines the interval used to query scribe for status
// updates. By default this is set to 5sec.
//
// Category maps a stream to a specific scribe category. You can define the
// wildcard stream (*) here, too. When set, all streams that do not have a
// specific mapping will go to this category (including _GOLLUM_).
// If no category mappings are set the stream name is used.
type Scribe struct {
	core.BufferedProducer `gollumdoc:embed_type`
	scribe            *scribe.ScribeClient
	transport         *thrift.TFramedTransport
	socket            *thrift.TSocket
	category          map[core.MessageStreamID]string
	batch             core.MessageBatch
	batchTimeout      time.Duration
	lastHeartBeat     time.Time
	heartBeatInterval time.Duration
	batchMaxCount     int
	batchFlushCount   int
	bufferSizeByte    int
	windowSize        int
	counters          map[string]*int64
	lastMetricUpdate  time.Time
}

const (
	scribeMetricMessages    = "Scribe:Messages-"
	scribeMetricMessagesSec = "Scribe:MessagesSec-"
	scribeMetricWindowSize  = "Scribe:WindowSize"
	scribeMaxRetries        = 30
	scribeMaxSleepTimeMs    = 3000
)

func init() {
	core.TypeRegistry.Register(Scribe{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Scribe) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)

	prod.SetStopCallback(prod.close)
	host := conf.GetString("Address", "localhost:1463")

	prod.batchMaxCount = conf.GetInt("Batch/MaxCount", 8192)
	prod.windowSize = prod.batchMaxCount
	prod.batchFlushCount = conf.GetInt("Batch/FlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("Batch/TimeoutSec", 5)) * time.Second
	prod.batch = core.NewMessageBatch(prod.batchMaxCount)
	prod.heartBeatInterval = time.Duration(conf.GetInt("HeartBeatIntervalSec", 5)) * time.Second

	prod.bufferSizeByte = conf.GetInt("ConnectionBufferSizeKB", 1<<10) << 10 // 1 MB
	prod.category = conf.GetStreamMap("Categories", "")

	// Initialize scribe connection

	var err error
	prod.socket, err = thrift.NewTSocketTimeout(host, prod.batchTimeout)
	conf.Errors.Push(err)

	prod.transport = thrift.NewTFramedTransport(prod.socket)
	binProtocol := thrift.NewTBinaryProtocol(prod.transport, false, false)
	prod.scribe = scribe.NewScribeClientProtocol(prod.transport, binProtocol, binProtocol)

	tgo.Metric.New(scribeMetricWindowSize)
	tgo.Metric.SetI(scribeMetricWindowSize, prod.windowSize)

	for _, category := range prod.category {
		metricName := scribeMetricMessages + category
		tgo.Metric.New(metricName)
		tgo.Metric.NewRate(metricName, scribeMetricMessagesSec+category, time.Second, 10, 3, true)
	}

	return conf.Errors.OrNil()
}

func (prod *Scribe) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *Scribe) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *Scribe) tryOpenConnection() bool {
	if !prod.transport.IsOpen() {
		if err := prod.transport.Open(); err != nil {
			prod.Log.Error.Print("Connection error:", err)
			return false // ### return, cannot connect ###
		}

		prod.socket.Conn().(bufferedConn).SetWriteBuffer(prod.bufferSizeByte)
		prod.lastHeartBeat = time.Now().Add(prod.heartBeatInterval)
		prod.Log.Note.Print("Connection opened")
	}

	if time.Since(prod.lastHeartBeat) < prod.heartBeatInterval {
		return true // ### return, assume alive ###
	}

	// Check status only when not sending, otherwise scribe gets confused
	prod.lastHeartBeat = time.Now()
	prod.batch.WaitForFlush(0)

	if status, err := prod.scribe.GetStatus(); err != nil {
		prod.Log.Error.Print("Status error:", err)
	} else {
		switch status {
		case fb303.FbStatus_DEAD:
			prod.Log.Warning.Print("Status reported as dead.")
		case fb303.FbStatus_STOPPING:
			prod.Log.Warning.Print("Status reported as stopping.")
		case fb303.FbStatus_STOPPED:
			prod.Log.Warning.Print("Status reported as stopped.")
		default:
			return true // ### return, all is well ###
		}
	}

	// TBD: health check? (ex-fuse breaker)
	prod.transport.Close()
	return false
}

func (prod *Scribe) sendBatch() {
	if prod.tryOpenConnection() {
		prod.batch.Flush(prod.transformMessages)
	} else if prod.IsStopping() {
		prod.batch.Flush(prod.tryFallbackForMessages)
	}
}

func (prod *Scribe) tryFallbackForMessages(messages []*core.Message) {
	for _, msg := range messages {
		prod.TryFallback(msg)
	}
}

func (prod *Scribe) transformMessages(messages []*core.Message) {
	logBuffer := make([]*scribe.LogEntry, len(messages))

	// Convert messages to scribe log format
	for idx, msg := range messages {
		currentMsg := msg.Clone()

		category, exists := prod.category[currentMsg.StreamID()]
		if !exists {
			if category, exists = prod.category[core.WildcardStreamID]; !exists {
				category = core.StreamRegistry.GetStreamName(currentMsg.StreamID())
			}
			metricName := scribeMetricMessages + category
			tgo.Metric.New(metricName)
			tgo.Metric.NewRate(metricName, scribeMetricMessagesSec+category, time.Second, 10, 3, true)
			prod.category[currentMsg.StreamID()] = category
		}

		logBuffer[idx] = &scribe.LogEntry{
			Category: category,
			Message:  string(currentMsg.Data()),
		}

		tgo.Metric.Inc(scribeMetricMessages + category)
	}

	// Try to send the whole batch.
	// If this fails, reduce the number of items send until sending succeeds.
	idxStart := 0
	for retryCount := 0; retryCount < scribeMaxRetries; retryCount++ {
		idxEnd := tmath.MinI(len(logBuffer), idxStart+prod.windowSize)
		resultCode, err := prod.scribe.Log(logBuffer[idxStart:idxEnd])

		if resultCode == scribe.ResultCode_OK {
			idxStart = idxEnd
			if idxStart < len(logBuffer) {
				retryCount = -1 // incremented to 0 after continue
				continue        // ### continue, data left to send ###
			}

			// Grow the window on success so we don't get stuck at 1
			if prod.windowSize < prod.batchMaxCount {
				prod.windowSize = tmath.MinI(prod.windowSize*2, prod.batchMaxCount)
			}

			return // ### return, success ###
		}

		if err != nil || resultCode != scribe.ResultCode_TRY_LATER {
			prod.Log.Error.Printf("Log error %d: %s", resultCode, err.Error())

			// TBD: health check? (ex-fuse breaker)
			prod.transport.Close() // reconnect

			prod.tryFallbackForMessages(messages[idxStart:])
			return // ### return, failure ###
		}

		prod.windowSize = tmath.MaxI(1, prod.windowSize/2)
		tgo.Metric.SetI(scribeMetricWindowSize, prod.windowSize)

		time.Sleep(time.Duration(scribeMaxSleepTimeMs/scribeMaxRetries) * time.Millisecond)
	}

	prod.Log.Error.Printf("Server seems to be busy")
	prod.tryFallbackForMessages(messages[idxStart:])
}

func (prod *Scribe) close() {
	defer func() {
		prod.transport.Close()
		prod.WorkerDone()
	}()

	prod.DefaultClose()
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
