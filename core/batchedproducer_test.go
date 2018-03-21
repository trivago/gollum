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
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
	"sync"
	"testing"
	"time"
)

type mockBatchedProducer struct {
	BatchedProducer
	onFlushFunc AssemblyFunc
}

func (prod *mockBatchedProducer) Configure(conf PluginConfigReader) {
}

func (prod *mockBatchedProducer) Produce(workers *sync.WaitGroup) {
	// start default BatchMessageLoop
	prod.BatchMessageLoop(workers, func() AssemblyFunc { return prod.onFlushFunc })
}

func getMockBatchedProducer() mockBatchedProducer {
	return mockBatchedProducer{
		BatchedProducer: BatchedProducer{
			DirectProducer: DirectProducer{
				SimpleProducer: SimpleProducer{
					control:         make(chan PluginControl),
					streams:         []MessageStreamID{},
					fallbackStream:  nil, //it must be set after registration of stream
					runState:        new(PluginRunState),
					modulators:      ModulatorArray{},
					shutdownTimeout: 10 * time.Millisecond,
					Logger:          logrus.WithField("Scope", "test"),
				},
			},
		},
	}
}

func TestBatchedProducerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBatchedProducer{}

	mockConf := NewPluginConfig("mockBatched", "mockBatchedProducer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("FallbackStream", "mockStream")

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("mockStream")

	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mockProducer)
	expect.NoError(err)
}

func TestBatchedProducerState(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBatchedProducer{}
	mockProducer.runState = new(PluginRunState)
	mockProducer.setState(PluginStateActive)
	expect.Equal(PluginStateActive, mockProducer.GetState())
	expect.True(mockProducer.IsActive())

	mockProducer.setState(PluginStateWaiting)
	expect.True(mockProducer.IsBlocked())

	mockProducer.setState(PluginStateStopping)
	expect.True(mockProducer.IsStopping())

	expect.True(mockProducer.IsActiveOrStopping())
}

func TestBatchedProducerCallback(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBatchedProducer{}
	rollBackCalled := false

	rollCallBack := func() {
		rollBackCalled = true
	}

	mockProducer.SetRollCallback(rollCallBack)
	mockProducer.onRoll()
	expect.True(rollBackCalled)

	stopCallBackCalled := false
	stopCallBack := func() {
		stopCallBackCalled = true
	}
	mockProducer.SetStopCallback(stopCallBack)
	mockProducer.onStop()
	expect.True(stopCallBackCalled)
}

func TestBatchedProducerEnqueue(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockP := getMockBatchedProducer()

	// configure and init producer
	mockConf := NewPluginConfig("BatchedProducerEnqueue", "mockBatchedProducer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("Batch/MaxCount", 3)
	mockConf.Override("Batch/FlushCount", 2)
	mockConf.Override("Batch/TimeoutSec", 1)

	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mockP)
	expect.NoError(err)

	// init test message
	msg := NewMessage(nil, []byte("BatchedProducerEnqueueTest"), nil, 1)

	// init flush func for test
	onBatchFlushExecutedGuard := sync.RWMutex{}
	onBatchFlushExecuted := false

	mockP.onFlushFunc = func(messages []*Message) {
		onBatchFlushExecutedGuard.Lock()
		onBatchFlushExecuted = true
		onBatchFlushExecutedGuard.Unlock()

		expect.Equal(2, len(messages)) // expect two messages because one is send during "stopping" state
		expect.Equal("BatchedProducerEnqueueTest", messages[0].String())
		expect.Equal("BatchedProducerEnqueueTest", messages[1].String())
	}

	// init WaitGroup and start produce method (loop)
	waitForTest := new(sync.WaitGroup)

	waitForTest.Add(1)
	go func() {
		defer waitForTest.Done()
		mockP.Produce(waitForTest)
	}()

	mockP.setState(PluginStateActive)

	// enqueue test messages
	mockP.Enqueue(msg, time.Second)
	mockP.Enqueue(msg, time.Second)

	// wait for flush
	time.Sleep(1500 * time.Millisecond)

	// expect execution of flush method
	onBatchFlushExecutedGuard.RLock()
	expect.Equal(true, onBatchFlushExecuted)
	onBatchFlushExecutedGuard.RUnlock()

	// stop producer
	mockP.Control() <- PluginControlStopProducer // trigger stopLoop (stop expect.NonBlocking)
	waitForTest.Wait()
}

func TestBatchedProducerClose(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockP := getMockBatchedProducer()

	// configure and init producer
	mockConf := NewPluginConfig("mockBatchedProducerClose", "mockBatchedProducer")
	mockConf.Override("Streams", []string{"testBoundStream"})

	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mockP)
	expect.NoError(err)

	// init test message
	msg := NewMessage(nil, []byte("BatchedProducerEnqueueTest"), nil, 1)

	// init flush func for test
	onBatchFlushExecuted := false
	mockP.onFlushFunc = func(messages []*Message) {
		onBatchFlushExecuted = true
		expect.Equal(2, len(messages)) // expect two messages because one is send during "stopping" state
		expect.Equal("BatchedProducerEnqueueTest", messages[0].String())
		expect.Equal("BatchedProducerEnqueueTest", messages[1].String())
	}

	// init WaitGroup and start produce method (loop)
	waitForTest := new(sync.WaitGroup)

	waitForTest.Add(1)
	go func() {
		defer waitForTest.Done()
		mockP.Produce(waitForTest)
	}()

	mockP.setState(PluginStateActive)

	// enqueue test messages
	mockP.Enqueue(msg, time.Second)
	mockP.Enqueue(msg, time.Second)

	// stop producer
	mockP.Control() <- PluginControlStopProducer // trigger stopLoop (stop expect.NonBlocking)
	waitForTest.Wait()
	time.Sleep(500 * time.Millisecond)

	// expect execution of flush method
	expect.Equal(true, onBatchFlushExecuted)
}
