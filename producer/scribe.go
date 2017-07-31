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

// Scribe producer
//
// This producer allows sending messages to Facebook's scribe service.
//
// Parameters
//
// - Address: Defines the host and port of a scrive endpoint.
// By default this parameter is set to "localhost:1463".
//
// - ConnectionBufferSizeKB: Sets the connection socket buffer size in KB.
// By default this parameter is set to 1024.
//
// - HeartBeatIntervalSec: Defines the interval in seconds used to query scribe
// for status updates.
// By default this parameter is set to 1.
//
// Category maps a stream to a scribe category. You can define the wildcard
// stream (*) here, too. When set, all streams that do not have a specific
// mapping will go to this category (including reserved streams like _GOLLUM_).
// If no category mappings are set the stream name is used as category.
// By default this parameter is set to an empty list.
//
// Examples
//
//  logs:
//    Type: producer.Scribe"
//    Stream: ["*", "_GOLLUM"]
//    Address: "scribe01:1463"
//	  HeartBeatIntervalSec: 10
//    Category:
//      "access"   : "accesslogs"
//      "error"    : "errorlogs"
//      "_GOLLUM_" : "gollumlogs"
type Scribe struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	scribe                *scribe.ScribeClient
	transport             *thrift.TFramedTransport
	socket                *thrift.TSocket
	category              map[core.MessageStreamID]string
	batch                 core.MessageBatch
	lastHeartBeat         time.Time
	bufferSizeByte        int           `config:"ConnectionBufferSizeKB" default:"1024" metric:"kb"`
	heartBeatInterval     time.Duration `config:"HeartBeatIntervalSec" default:"5" metric:"sec"`
	batchTimeout          time.Duration `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	batchMaxCount         int           `config:"Batch/MaxCount" default:"8192"`
	batchFlushCount       int           `config:"Batch/FlushCount" default:"4096"`
	windowSize            int
	counters              map[string]*int64
	lastMetricUpdate      time.Time
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
func (prod *Scribe) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)
	host := conf.GetString("Address", "localhost:1463")

	prod.category = conf.GetStreamMap("Categories", "")
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.windowSize = prod.batchMaxCount
	prod.batch = core.NewMessageBatch(prod.batchMaxCount)

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
			prod.Logger.Error("Connection error:", err)
			return false // ### return, cannot connect ###
		}

		prod.socket.Conn().(bufferedConn).SetWriteBuffer(prod.bufferSizeByte)
		prod.lastHeartBeat = time.Now().Add(prod.heartBeatInterval)
		prod.Logger.Info("Connection opened")
	}

	if time.Since(prod.lastHeartBeat) < prod.heartBeatInterval {
		return true // ### return, assume alive ###
	}

	// Check status only when not sending, otherwise scribe gets confused
	prod.lastHeartBeat = time.Now()
	prod.batch.WaitForFlush(0)

	if status, err := prod.scribe.GetStatus(); err != nil {
		prod.Logger.Error("Status error:", err)
	} else {
		switch status {
		case fb303.FbStatus_DEAD:
			prod.Logger.Warning("Status reported as dead.")
		case fb303.FbStatus_STOPPING:
			prod.Logger.Warning("Status reported as stopping.")
		case fb303.FbStatus_STOPPED:
			prod.Logger.Warning("Status reported as stopped.")
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
		category, exists := prod.category[msg.GetStreamID()]
		if !exists {
			if category, exists = prod.category[core.WildcardStreamID]; !exists {
				category = core.StreamRegistry.GetStreamName(msg.GetStreamID())
			}
			metricName := scribeMetricMessages + category
			tgo.Metric.New(metricName)
			tgo.Metric.NewRate(metricName, scribeMetricMessagesSec+category, time.Second, 10, 3, true)
			prod.category[msg.GetStreamID()] = category
		}

		logBuffer[idx] = &scribe.LogEntry{
			Category: category,
			Message:  string(msg.GetPayload()),
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
			prod.Logger.Errorf("Log error %d: %s", resultCode, err.Error())

			// TBD: health check? (ex-fuse breaker)
			prod.transport.Close() // reconnect

			prod.tryFallbackForMessages(messages[idxStart:])
			return // ### return, failure ###
		}

		prod.windowSize = tmath.MaxI(1, prod.windowSize/2)
		tgo.Metric.SetI(scribeMetricWindowSize, prod.windowSize)

		time.Sleep(time.Duration(scribeMaxSleepTimeMs/scribeMaxRetries) * time.Millisecond)
	}

	prod.Logger.Errorf("Server seems to be busy")
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
