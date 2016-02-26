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
	"github.com/trivago/gollum/shared"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockProducer struct {
	ProducerBase
}

func (prod *mockProducer) Produce(workers *sync.WaitGroup) {
	// does something.
}

func getMockProducer() mockProducer {
	return mockProducer{
		ProducerBase{
			messages:        make(chan Message, 2),
			control:         make(chan PluginControl),
			streams:         []MessageStreamID{},
			dropStreamID:    2,
			runState:        new(PluginRunState),
			timeout:         500 * time.Millisecond,
			filter:          &mockFilter{},
			format:          &mockFormatter{},
			shutdownTimeout: 10 * time.Millisecond,
			fuseName:        "test",
			fuseTimeout:     100 * time.Millisecond,
		},
	}
}

func TestProducerConfigure(t *testing.T) {
	expect := shared.NewExpect(t)

	mockProducer := mockProducer{}

	shared.TypeRegistry.Register(mockPlugin{})
	shared.TypeRegistry.Register(mockFormatter{})
	shared.TypeRegistry.Register(mockFilter{})
	mockConf := NewPluginConfig("core.mockPlugin")
	mockConf.ID = "testPluginConf"
	mockConf.Stream = []string{"testBoundStream"}

	err := mockProducer.Configure(mockConf)
	expect.NotNil(err)
	mockConf.Settings["Formatter"] = "core.mockFormatter"

	err = mockProducer.Configure(mockConf)
	expect.NotNil(err)
	mockConf.Settings["Filter"] = "core.mockFilter"

	err = mockProducer.Configure(mockConf)
	expect.Nil(err)
}

func TestProducerState(t *testing.T) {
	expect := shared.NewExpect(t)

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
	expect := shared.NewExpect(t)

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

func TestProducerDependency(t *testing.T) {
	expect := shared.NewExpect(t)
	mockP := mockProducer{}

	secondMockP := mockProducer{}
	thirdMockP := mockProducer{}

	// give dropStreamId to differentiate secondMockP and thirdMockP
	secondMockP.dropStreamID = 1
	thirdMockP.dropStreamID = 2

	secondMockP.AddDependency(&thirdMockP)
	mockP.dependencies = []Producer{}
	mockP.AddDependency(&secondMockP)

	//add secondMockP again. Shouldn't add it in its dependency list
	mockP.AddDependency(&secondMockP)
	expect.Equal(1, len(mockP.dependencies))

	expect.True(mockP.DependsOn(&secondMockP))
	expect.True(mockP.DependsOn(&thirdMockP))
}

func TestProducerEnqueue(t *testing.T) {
	// TODO: distribute for drop route not called. Probably streams array contains soln
	expect := shared.NewExpect(t)
	mockP := getMockProducer()
	mockDistribute := func(msg Message) {
		expect.Equal("ProdEnqueueTest", msg.String())
	}
	mockDropStream := getMockStream()
	StreamRegistry.Register(&mockDropStream, 2)
	mockDropStream.distribute = mockDistribute

	msg := Message{
		Data:     []byte("ProdEnqueueTest"),
		StreamID: 1,
	}
	enqTimeout := time.Second
	mockP.setState(PluginStateStopping)
	// cause panic and check if message is dropped
	mockP.Enqueue(msg, &enqTimeout)

	mockP.setState(PluginStateActive)
	mockP.Enqueue(msg, &enqTimeout)

	mockStream := getMockStream()
	mockStream.distribute = mockDistribute
	StreamRegistry.Register(&mockStream, 1)

	go func() {
		mockP.Enqueue(msg, &enqTimeout)
	}()
	//give time for message to enqueue in the channel
	time.Sleep(200 * time.Millisecond)

	ret := <-mockP.messages
	expect.Equal("ProdEnqueueTest", ret.String())

}

func TestProducerCloseMessageChannel(t *testing.T) {
	expect := shared.NewExpect(t)
	mockP := getMockProducer()

	mockP.setState(PluginStateActive)

	handleMessageFail := func(msg Message) {
		time.Sleep(20 * time.Millisecond)
	}

	handleMessage := func(msg Message) {
		expect.Equal("closeMessageChannel", msg.String())
	}

	mockDistribute := func(msg Message) {
		expect.Equal("closeMessageChannel", msg.String())
	}
	mockDropStream := getMockStream()
	mockDropStream.distribute = mockDistribute
	mockDropStream.AddProducer(&mockProducer{})

	StreamRegistry.name[2] = "testStream"
	StreamRegistry.Register(&mockDropStream, 2)

	mockP.streams = []MessageStreamID{2}
	msgToSend := Message{
		Data:     []byte("closeMessageChannel"),
		StreamID: 1,
	}
	mockP.messages <- msgToSend
	mockP.messages <- msgToSend
	ret := mockP.CloseMessageChannel(handleMessageFail)
	expect.False(ret)

	mockP.messages = make(chan Message, 2)
	mockP.messages <- msgToSend
	ret = mockP.CloseMessageChannel(handleMessage)
	expect.True(ret)

}

func TestProducerTickerLoop(t *testing.T) {
	expect := shared.NewExpect(t)
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
	expect := shared.NewExpect(t)
	mockP := getMockProducer()
	mockP.setState(PluginStateActive)
	mockP.messages = make(chan Message, 10)
	msgData := "test Message loop"
	msg := Message{
		Data: []byte(msgData),
	}

	for i := 0; i < 9; i++ {
		mockP.messages <- msg
	}
	counter := 0
	onMessage := func(msg Message) {
		expect.Equal(msgData, msg.String())
		counter++
		if counter == 9 {
			mockP.setState(PluginStateDead)
		}
	}

	mockP.messageLoop(onMessage)
	expect.Equal(9, counter)
}

func TestProducerWaitForDependencies(t *testing.T) {
	expect := shared.NewExpect(t)
	mockP := getMockProducer()

	for i := 0; i < 5; i++ {
		dep := getMockProducer()
		dep.setState(PluginStateActive)
		mockP.AddDependency(&dep)
	}
	routine := func() {
		mockP.WaitForDependencies(PluginStateStopping, 50*time.Millisecond)
	}

	go expect.NonBlocking(2*time.Second, routine)

	// Resolve states so that expect.NonBlocking returns
	for _, dep := range mockP.dependencies {
		ped := dep.(*mockProducer)
		ped.setState(PluginStateDead)
	}
}

func TestProducerControlLoop(t *testing.T) {
	expect := shared.NewExpect(t)
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
	expect := shared.NewExpect(t)
	activateFuse := new(int32)
	checkCounter := new(int32)

	mockP := getMockProducer()
	mockP.SetCheckFuseCallback(func() bool {
		atomic.AddInt32(checkCounter, 1)
		return atomic.LoadInt32(activateFuse) == 1
	})

	fuse := StreamRegistry.GetFuse(mockP.fuseName)
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
