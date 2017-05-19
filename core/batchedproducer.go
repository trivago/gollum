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

package core

import (
	"github.com/trivago/tgo/tmath"
	"sync"
	"time"
)

// BatchedProducer plugin base type
// This type defines a common BatchedProducer base class. Producers may
// derive from this class.
//
// Configuration example:
//
//  - producer.Foobar:
//      Streams:
//        - "foo"
//        - "bar"
//      Enable: true
//      ShutdownTimeoutMs: 1000
//      FallbackStream: "_DROPPED_"
//      Batch:
//        MaxCount: 8192
//    	  FlushCount: 4096
//    	  TimeoutSec: 5
//
type BatchedProducer struct {
	DirectProducer  `gollumdoc:"embed_type"`
	Batch           MessageBatch
	batchTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
	onBatchFlush    func() AssemblyFunc
}

// Configure initializes the standard producer config values.
func (prod *BatchedProducer) Configure(conf PluginConfigReader) error {
	prod.DirectProducer.Configure(conf)

	prod.batchMaxCount = conf.GetInt("Batch/MaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("Batch/FlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("Batch/TimeoutSec", 5)) * time.Second

	prod.Batch = NewMessageBatch(prod.batchMaxCount)

	return conf.Errors.OrNil()
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop. A timeout value != nil will overwrite the channel
// timeout value for this call.
func (prod *BatchedProducer) Enqueue(msg *Message, timeout *time.Duration) {
	defer prod.enqueuePanicHandling(msg)

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.TryFallback(msg)
		return // ### return, closing down ###
	}

	if prod.HasContinueAfterModulate(msg) == false {
		return
	}

	prod.appendMessage(msg)
}

// appendMessage append a message to the batch at enqueuing
func (prod *BatchedProducer) appendMessage(msg *Message) {
	prod.Batch.AppendOrFlush(msg, prod.flushBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

// flushBatch is the used function pointer to flush the batch
func (prod *BatchedProducer) flushBatch() {
	prod.Batch.Flush(prod.onBatchFlush())
}

// flushBatchOnTimeOut is the used function pointer to flush the batch on timeout or reached max size
func (prod *BatchedProducer) flushBatchOnTimeOut() {
	if prod.Batch.ReachedTimeThreshold(prod.batchTimeout) || prod.Batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.flushBatch()
	}
}

// BatchMessageLoop start the TickerMessageControlLoop() for batch producer
func (prod *BatchedProducer) BatchMessageLoop(workers *sync.WaitGroup, onBatchFlush func() AssemblyFunc) {
	prod.onBatchFlush = onBatchFlush

	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.appendMessage, prod.batchTimeout, prod.flushBatchOnTimeOut)
}
