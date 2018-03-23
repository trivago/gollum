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

package core

import (
	"sync"
	"time"

	"github.com/trivago/tgo/tmath"
)

// BatchedProducer producer
//
// This type defines a common base type that can be used by producers that work
// better when sending batches of messages instead of single ones. Batched
// producers will collect messages and flush them after certain limits have
// been reached. The flush process will be done in the background. Message
// collection continues non-blocking unless flushing takes longer than filling
// up the internal buffer again.
//
// Parameters
//
// - Batch/MaxCount: Defines the maximum number of messages per batch. If this
// limit is reached a flush is always triggered.
// By default this parameter is set to 8192.
//
// - Batch/FlushCount: Defines the minimum number of messages required to flush
// a batch. If this limit is reached a flush might be triggered.
// By default this parameter is set to 4096.
//
// - Batch/TimeoutSec: Defines the maximum time in seconds messages can stay in
// the internal buffer before being flushed.
// By default this parameter is set to 5.
type BatchedProducer struct {
	DirectProducer  `gollumdoc:"embed_type"`
	Batch           MessageBatch
	batchMaxCount   int           `config:"Batch/MaxCount" default:"8192"`
	batchFlushCount int           `config:"Batch/FlushCount" default:"4096"`
	batchTimeout    time.Duration `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	onBatchFlush    func() AssemblyFunc
}

// Configure initializes the standard producer config values.
func (prod *BatchedProducer) Configure(conf PluginConfigReader) {
	prod.SetStopCallback(prod.DefaultClose)

	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.Batch = NewMessageBatch(prod.batchMaxCount)
}

// Enqueue will add the message to the internal channel so it can be processed
// by the producer main loop. A timeout value != nil will overwrite the channel
// timeout value for this call.
func (prod *BatchedProducer) Enqueue(msg *Message, timeout time.Duration) {
	defer prod.enqueuePanicHandling(msg)

	// Don't accept messages if we are shutting down
	if prod.GetState() >= PluginStateStopping {
		prod.TryFallback(msg)
		return // ### return, closing down ###
	}

	if !prod.HasContinueAfterModulate(msg) {
		return
	}

	prod.appendMessage(msg)
	MessageTrace(msg, prod.GetID(), "Enqueued by batched producer")
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

// DefaultClose defines the default closing process
func (prod *BatchedProducer) DefaultClose() {
	defer prod.WorkerDone()
	prod.Batch.Close(prod.onBatchFlush(), prod.GetShutdownTimeout())
}
