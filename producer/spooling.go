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
	"github.com/trivago/gollum/shared"
	"runtime"
	"sync"
	"time"
)

// Spooling producer plugin
// Configuration example
//
//   - "producer.Spooling":
//     Enable: true
//     RetryDelayMs: 2000
//     MaxMessageCount: 0
//
// The Spooling producer buffers messages and sends them again to the previous
// stream stored in the message. This means the message must have been routed
// at least once before reaching the spooling producer. If the previous and
// current stream is identical the message is dropped.
//
// RetryDelayMs denotes the number of milliseconds before a message is send
// again. The message is removed from the list after sending.
// If the time is set to 0, messages are not buffered but resent directly.
// By default this is set to 2000 (2 seconds).
//
// MaxMessageCount denotes the maximum number of messages to store before
// dropping new incoming messages. By default this is set to 0 which means all
// messages are stored.
type Spooling struct {
	core.ProducerBase
	retryDuration   time.Duration
	maxMessageCount int
	bufferGuard     *sync.Mutex
	messageBuffer   []scheduledMessage
	closing         bool
	metricName      string
}

type scheduledMessage struct {
	incoming time.Time
	message  core.Message
}

func init() {
	shared.RuntimeType.Register(Spooling{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Spooling) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.SetPrepareStopCallback(func() { prod.closing = true })

	prod.retryDuration = time.Duration(conf.GetInt("RetryDelayMs", 2000)) * time.Millisecond
	prod.maxMessageCount = conf.GetInt("MaxMessageCount", 0)
	prod.bufferGuard = new(sync.Mutex)
	prod.closing = false
	prod.metricName = "spoolsize_" + core.StreamTypes.GetStreamName(prod.Streams()[0])

	shared.Metric.New(prod.metricName)

	return nil
}

func newScheduledMessage(msg core.Message) scheduledMessage {
	return scheduledMessage{
		incoming: time.Now(),
		message:  msg,
	}
}

func (prod *Spooling) storeMessage(msg core.Message) {
	if prod.maxMessageCount <= 0 || len(prod.messageBuffer) < prod.maxMessageCount {
		prod.bufferGuard.Lock()
		defer prod.bufferGuard.Unlock()
		prod.messageBuffer = append(prod.messageBuffer, newScheduledMessage(msg))

		shared.Metric.SetI(prod.metricName, len(prod.messageBuffer))
	} else {
		prod.Drop(msg)
	}
}

func (prod *Spooling) sendMessage(msg core.Message) {
	buffer := make([]byte, 1024)
	runtime.Stack(buffer, false)

	if !prod.closing && msg.StreamID != msg.PrevStreamID {
		msg.Route(msg.PrevStreamID)
	} else {
		prod.Drop(msg)
	}
}

func (prod *Spooling) buildFlushList() []core.Message {
	prod.bufferGuard.Lock()
	defer prod.bufferGuard.Unlock()

	flushList := []core.Message{}
	sliceIdx := 0

	for sliceIdx < len(prod.messageBuffer) {
		message := prod.messageBuffer[sliceIdx]
		if time.Since(message.incoming) < prod.retryDuration {
			break // ### break, found all messages ###
		}
		flushList = append(flushList, message.message)
		sliceIdx++
	}
	prod.messageBuffer = prod.messageBuffer[sliceIdx:]
	return flushList
}

func (prod *Spooling) startFlushWorker() {
	if !prod.closing && prod.retryDuration > 0 {
		time.AfterFunc(prod.retryDuration, prod.flushWorker)
	}
}

func (prod *Spooling) flushWorker() {
	defer prod.startFlushWorker() // because resend may block

	flushList := prod.buildFlushList()
	for _, msg := range flushList {
		prod.sendMessage(msg)
	}
}

// Close gracefully
func (prod *Spooling) Close() {
	defer prod.WorkerDone()
	prod.closing = true
	prod.CloseGracefully(prod.sendMessage)
	for _, message := range prod.messageBuffer {
		prod.Drop(message.message)
	}
}

// Produce writes to stdout or stderr.
func (prod *Spooling) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.startFlushWorker()

	if prod.retryDuration > 0 {
		prod.DefaultControlLoop(prod.storeMessage)
	} else {
		prod.DefaultControlLoop(prod.sendMessage)
	}
}
