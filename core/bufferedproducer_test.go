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
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type mockBufferedProducer struct {
	BufferedProducer
}

func (prod *mockBufferedProducer) Configure(conf PluginConfigReader) {
}

func (prod *mockBufferedProducer) Produce(workers *sync.WaitGroup) {
	// does something.
}

func getMockBufferedProducer() mockBufferedProducer {
	return mockBufferedProducer{
		BufferedProducer{
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
			messages:       NewMessageQueue(2),
			channelTimeout: 500 * time.Millisecond,
		},
	}
}

func TestProducerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBufferedProducer{}

	mockConf := NewPluginConfig("mock", "mockBufferedProducer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("FallbackStream", "mockStream")

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("mockStream")

	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mockProducer)
	expect.NoError(err)
}

func TestProducerState(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBufferedProducer{}
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

func TestProducerCallback(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockBufferedProducer{}
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

func TestProducerWaitgroup(t *testing.T) {
	// TODO: Well, complete this obviously

}

func TestProducerEnqueue(t *testing.T) {
	// TODO: distribute for drop route not called. Probably routers array contains soln
	expect := ttesting.NewExpect(t)
	mockP := getMockBufferedProducer()

	mockDropStream := getMockRouter()
	mockDropStream.streamID = 2
	StreamRegistry.Register(&mockDropStream, 2)

	mockP.fallbackStream = StreamRegistry.GetRouter(2)

	msg := NewMessage(nil, []byte("ProdEnqueueTest"), nil, 1)

	enqTimeout := time.Second
	mockP.setState(PluginStateStopping)
	// cause panic and check if message is sent to the fallback
	mockP.Enqueue(msg, enqTimeout)

	mockP.setState(PluginStateActive)
	mockP.Enqueue(msg, enqTimeout)

	mockStream := getMockRouter()
	mockDropStream.streamID = 1
	StreamRegistry.Register(&mockStream, 1)

	go func() {
		mockP.Enqueue(msg, enqTimeout)
	}()
	//give time for message to enqueue in the channel
	time.Sleep(200 * time.Millisecond)

	ret, _ := mockP.messages.Pop()
	expect.Equal("ProdEnqueueTest", ret.String())

}

func TestProducerCloseMessageChannel(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockBufferedProducer()

	mockP.setState(PluginStateActive)

	handleMessageFail := func(msg *Message) {
		expect.Equal("closeMessageChannel", msg.String())
		time.Sleep(mockP.GetShutdownTimeout() * 10)
	}

	handleMessage := func(msg *Message) {
		expect.Equal("closeMessageChannel", msg.String())
	}

	mockDropStream := getMockRouter()
	mockDropStream.AddProducer(&mockBufferedProducer{})
	mockDropStream.streamID = 2

	StreamRegistry.name[2] = "testStream"
	StreamRegistry.Register(&mockDropStream, 2)

	mockP.streams = []MessageStreamID{2}
	msgToSend := NewMessage(nil, []byte("closeMessageChannel"), nil, 2)

	mockP.Enqueue(msgToSend, time.Duration(0))
	mockP.Enqueue(msgToSend, time.Duration(0))
	mockP.CloseMessageChannel(handleMessageFail)

	mockP.messages = NewMessageQueue(2)
	mockP.Enqueue(msgToSend, time.Duration(0))
	mockP.CloseMessageChannel(handleMessage)
}

func TestProducerTickerLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockBufferedProducer()
	mockP.setState(PluginStateActive)
	// accept timeroff by abs( 15ms)
	delta := float64(15 * time.Millisecond)
	counter := new(int32)
	tickerLoopTimeout := 20 * time.Millisecond
	var timeRecorded time.Time
	onTimeOut := func() {
		if atomic.LoadInt32(counter) > 3 {
			mockP.setState(PluginStateDead)
			return
		}
		//this was fired as soon as the ticker started. So ignore but save the time
		if atomic.LoadInt32(counter) == 0 {
			timeRecorded = time.Now()
			atomic.AddInt32(counter, 1)
			return
		}
		diff := time.Since(timeRecorded)
		deltaDiff := math.Abs(float64(tickerLoopTimeout - diff))
		expect.Less(deltaDiff, delta)
		timeRecorded = time.Now()
		atomic.AddInt32(counter, 1)
	}

	mockP.tickerLoop(tickerLoopTimeout, onTimeOut)
	time.Sleep(2 * time.Second)
	// in anycase, the callback has to be called atleast once
	expect.Greater(atomic.LoadInt32(counter), int32(1))
}

func TestProducerMessageLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockBufferedProducer()
	mockP.setState(PluginStateActive)
	mockP.messages = NewMessageQueue(10)
	msgData := "test Message loop"
	msg := new(Message)
	msg.data.payload = []byte(msgData)

	for i := 0; i < 9; i++ {
		mockP.messages.Push(msg, -1)
	}
	counter := 0
	onMessage := func(msg *Message) {
		expect.Equal(msgData, msg.String())
		counter++
		if counter == 9 {
			mockP.setState(PluginStateDead)
		}
	}

	mockP.messageLoop(onMessage)
	expect.Equal(9, counter)
}

func TestProducerControlLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockBufferedProducer()

	stop := new(int32)
	roll := new(int32)
	mockP.onStop = func() {
		atomic.StoreInt32(stop, 1)
	}

	mockP.onRoll = func() {
		atomic.StoreInt32(roll, 1)
	}

	waitForTest := new(sync.WaitGroup)
	waitForTest.Add(1)
	go func() {
		defer waitForTest.Done()
		expect.NonBlocking(2*time.Second, mockP.ControlLoop)
	}()

	mockP.Control() <- PluginControlStopProducer // trigger stopLoop (stop expect.NonBlocking)
	waitForTest.Wait()
	expect.Equal(atomic.LoadInt32(stop), int32(1))

	waitForTest.Add(1)
	go func() {
		defer waitForTest.Done()
		expect.NonBlocking(2*time.Second, mockP.ControlLoop)
	}()

	mockP.Control() <- PluginControlRoll // trigger rollLoop
	mockP.Control() <- PluginControlStopProducer

	waitForTest.Wait()
	expect.Equal(atomic.LoadInt32(roll), int32(1))

}
