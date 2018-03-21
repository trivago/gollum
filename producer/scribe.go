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
	"fmt"
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
// - WindowSize: Defines the maximum number of messages send to scribe in one
// call. The WindowSize will reduce when scribe is returing "try later" to
// reduce load on the scribe server. It will slowly rise again for each
// successful write until WindowSize is reached again.
// By default this parameter is set to 2048.
//
// - ConnectionTimeoutSec: Defines the time in seconds after which a connection
// timeout is assumed. This can happen during writes or status reads.
// By default this parameter is set to 5.
//
// - Category: Maps a stream to a scribe category. You can define the wildcard
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
//    HeartBeatIntervalSec: 10
//    Category:
//      "access"   : "accesslogs"
//      "error"    : "errorlogs"
//      "_GOLLUM_" : "gollumlogs"
type Scribe struct {
	core.BatchedProducer `gollumdoc:"embed_type"`
	scribe               *scribe.ScribeClient
	transport            *thrift.TFramedTransport
	socket               *thrift.TSocket
	categoryGuard        *sync.RWMutex
	category             map[core.MessageStreamID]string
	lastHeartBeat        time.Time
	windowSize           int
	bufferSizeByte       int           `config:"ConnectionBufferSizeKB" default:"1024" metric:"kb"`
	heartBeatInterval    time.Duration `config:"HeartBeatIntervalSec" default:"5" metric:"sec"`
	maxWindowSize        int           `config:"WindowSize" default:"2048"`
	connectionTimeout    time.Duration `config:"ConnectionTimeoutSec" default:"5" metric:"sec"`
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
	prod.windowSize = prod.maxWindowSize
	prod.categoryGuard = new(sync.RWMutex)

	// Initialize scribe connection

	var err error
	prod.socket, err = thrift.NewTSocketTimeout(host, prod.connectionTimeout)
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
	prod.lastHeartBeat = time.Now()

	// Check status only when not sending, otherwise scribe gets confused
	err := prod.Batch.AfterFlushDo(func() error {
		status, err := prod.scribe.GetStatus()
		if err != nil {
			return err
		}
		switch status {
		case fb303.FbStatus_DEAD:
			return fmt.Errorf("Service dead")
		case fb303.FbStatus_STOPPING:
			return fmt.Errorf("Service stopping")
		case fb303.FbStatus_STOPPED:
			return fmt.Errorf("Service stopped")
		}
		return nil // ### return, all is well ###
	})

	if err == nil {
		return true
	}

	prod.Logger.WithError(err).Error("Scribe status check failed")
	prod.transport.Close()
	return false
}

func (prod *Scribe) sendBatch() core.AssemblyFunc {
	if prod.tryOpenConnection() {
		return prod.transformMessages
	}
	if prod.IsStopping() {
		return prod.tryFallbackForMessages
	}
	return nil
}

func (prod *Scribe) tryFallbackForMessages(messages []*core.Message) {
	for _, msg := range messages {
		prod.TryFallback(msg)
	}
}

func (prod *Scribe) addCategory(streamID core.MessageStreamID) string {
	prod.categoryGuard.Lock()
	defer prod.categoryGuard.Unlock()

	category, exists := prod.category[core.WildcardStreamID]
	if exists {
		return category
	}

	category = core.StreamRegistry.GetStreamName(streamID)

	metricName := scribeMetricMessages + category
	tgo.Metric.New(metricName)
	tgo.Metric.NewRate(metricName, scribeMetricMessagesSec+category, time.Second, 10, 3, true)

	prod.category[streamID] = category
	return category
}

func (prod *Scribe) transformMessages(messages []*core.Message) {
	logBuffer := make([]*scribe.LogEntry, len(messages))

	// Convert messages to scribe log format
	for idx, msg := range messages {
		prod.categoryGuard.RLock()
		category, exists := prod.category[msg.GetStreamID()]
		prod.categoryGuard.RUnlock()

		if !exists {
			category = prod.addCategory(msg.GetStreamID())
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
			if prod.windowSize < prod.maxWindowSize {
				prod.windowSize = tmath.MinI(prod.windowSize*2, prod.maxWindowSize)
			}

			return // ### return, success ###
		}

		if err != nil || resultCode != scribe.ResultCode_TRY_LATER {
			prod.Logger.WithError(err).Errorf("Scribe error %d", resultCode)

			prod.transport.Close() // reconnect
			prod.tryFallbackForMessages(messages[idxStart:])
			return // ### return, failure ###
		}

		// Scribe said "try again".
		// Reduce window size to reduce load on server

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
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.BatchMessageLoop(workers, prod.sendBatch)
}
