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
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/ttesting"
)

type mockProducer struct {
	BufferedProducer
}

func (prod *mockProducer) Produce(workers *sync.WaitGroup) {
	// does something.
}

func getMockProducer() mockProducer {
	return mockProducer{
		BufferedProducer{
			SimpleProducer: SimpleProducer{
				control:          make(chan PluginControl),
				streams:          []MessageStreamID{},
				dropStream:       nil, //it must be set after registration of stream
				runState:         new(PluginRunState),
				modulators:       ModulatorArray{},
				shutdownTimeout:  10 * time.Millisecond,
				fuseName:         "test",
				fuseTimeout:      100 * time.Millisecond,
				fuseControlGuard: new(sync.Mutex),
				Log:              tlog.NewLogScope("test"),
			},
			messages:       NewMessageQueue(2),
			channelTimeout: 500 * time.Millisecond,
		},
	}
}

func TestProducerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockProducer{}

	mockConf := NewPluginConfig("", "mockProducer")
	mockConf.Override("streams", []string{"testBoundStream"})
	mockConf.Override("DropToStream", "mockStream")

	// Stream needs to be configured to avoid unknown class errors
	registerMockStream("mockStream")

	err := mockProducer.Configure(NewPluginConfigReader(&mockConf))
	expect.NoError(err)
}

func TestProducerState(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockProducer := mockProducer{}
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

	mockProducer := mockProducer{}
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
	// TODO: distribute for drop route not called. Probably streams array contains soln
	expect := ttesting.NewExpect(t)
	mockP := getMockProducer()

	mockDropStream := getMockStream()
	mockDropStream.streamID = 2
	StreamRegistry.Register(&mockDropStream, 2)

	mockP.dropStream = StreamRegistry.GetStream(2)

	msg := NewMessage(nil, []byte("ProdEnqueueTest"), 4, 1)

	enqTimeout := time.Second
	mockP.setState(PluginStateStopping)
	// cause panic and check if message is dropped
	mockP.Enqueue(msg, &enqTimeout)

	mockP.setState(PluginStateActive)
	mockP.Enqueue(msg, &enqTimeout)

	mockStream := getMockStream()
	mockDropStream.streamID = 1
	StreamRegistry.Register(&mockStream, 1)

	go func() {
		mockP.Enqueue(msg, &enqTimeout)
	}()
	//give time for message to enqueue in the channel
	time.Sleep(200 * time.Millisecond)

	ret, _ := mockP.messages.Pop()
	expect.Equal("ProdEnqueueTest", ret.String())

}

func TestProducerCloseMessageChannel(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockProducer()

	mockP.setState(PluginStateActive)

	handleMessageFail := func(msg *Message) {
		expect.Equal("closeMessageChannel", msg.String())
		time.Sleep(mockP.GetShutdownTimeout() * 10)
	}

	handleMessage := func(msg *Message) {
		expect.Equal("closeMessageChannel", msg.String())
	}

	mockDropStream := getMockStream()
	mockDropStream.AddProducer(&mockProducer{})
	mockDropStream.streamID = 2

	StreamRegistry.name[2] = "testStream"
	StreamRegistry.Register(&mockDropStream, 2)

	mockP.streams = []MessageStreamID{2}
	msgToSend := NewMessage(nil, []byte("closeMessageChannel"), 4, 2)

	mockP.Enqueue(msgToSend, nil)
	mockP.Enqueue(msgToSend, nil)
	mockP.CloseMessageChannel(handleMessageFail)

	mockP.messages = NewMessageQueue(2)
	mockP.Enqueue(msgToSend, nil)
	mockP.CloseMessageChannel(handleMessage)
}

func TestProducerTickerLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockProducer()
	mockP.setState(PluginStateActive)
	// accept timeroff by abs( 8 ms)
	delta := float64(8 * time.Millisecond)
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
		diff := time.Now().Sub(timeRecorded)
		deltaDiff := math.Abs(float64(tickerLoopTimeout - diff))
		expect.True(deltaDiff < delta)
		timeRecorded = time.Now()
		atomic.AddInt32(counter, 1)
		return
	}

	mockP.tickerLoop(tickerLoopTimeout, onTimeOut)
	time.Sleep(2 * time.Second)
	// in anycase, the callback has to be called atleast once
	expect.Greater(atomic.LoadInt32(counter), int32(1))
}

func TestProducerMessageLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockP := getMockProducer()
	mockP.setState(PluginStateActive)
	mockP.messages = NewMessageQueue(10)
	msgData := "test Message loop"
	msg := &Message{
		data: []byte(msgData),
	}

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
	mockP := getMockProducer()

	stop := new(int32)
	roll := new(int32)
	mockP.onStop = func() {
		atomic.StoreInt32(stop, 1)
	}

	mockP.onRoll = func() {
		atomic.StoreInt32(roll, 1)
	}

	go expect.NonBlocking(2*time.Second, mockP.ControlLoop)
	time.Sleep(50 * time.Millisecond)
	mockP.control <- PluginControlStopProducer // trigger stopLoop (stop expect.NonBlocking)
	time.Sleep(50 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(stop), int32(1))

	go expect.NonBlocking(2*time.Second, mockP.ControlLoop)
	time.Sleep(50 * time.Millisecond)
	mockP.control <- PluginControlRoll // trigger rollLoop (stop expect.NonBlocking)
	time.Sleep(50 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(roll), int32(1))

}

func TestProducerFuse(t *testing.T) {
	expect := ttesting.NewExpect(t)
	activateFuse := new(int32)
	checkCounter := new(int32)

	mockP := getMockProducer()
	mockP.SetCheckFuseCallback(func() bool {
		atomic.AddInt32(checkCounter, 1)
		return atomic.LoadInt32(activateFuse) == 1
	})

	fuse := FuseRegistry.GetFuse(mockP.fuseName)
	expect.False(fuse.IsBurned())

	go mockP.ControlLoop()

	// Check basic functionality

	expect.NonBlocking(time.Second, func() { mockP.Control() <- PluginControlFuseBurn })
	time.Sleep(mockP.fuseTimeout)
	expect.True(fuse.IsBurned())

	time.Sleep(mockP.fuseTimeout * 2)
	expect.True(fuse.IsBurned())
	expect.Greater(atomic.LoadInt32(checkCounter), int32(0))

	atomic.StoreInt32(activateFuse, 1)
	time.Sleep(mockP.fuseTimeout * 2)
	expect.False(fuse.IsBurned())

	// Check double calls

	atomic.StoreInt32(activateFuse, 0)
	expect.NonBlocking(time.Second, func() { mockP.Control() <- PluginControlFuseBurn })
	expect.NonBlocking(time.Second, func() { mockP.Control() <- PluginControlFuseBurn })
	expect.True(fuse.IsBurned())

	expect.NonBlocking(time.Second, func() { mockP.Control() <- PluginControlFuseActive })
	expect.NonBlocking(time.Second, func() { mockP.Control() <- PluginControlFuseActive })
	expect.False(fuse.IsBurned())
}
