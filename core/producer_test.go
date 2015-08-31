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

package core

import (
	"github.com/trivago/gollum/shared"
	"sync"
	"testing"
	"time"
)

type mockProducer struct {
	ProducerBase
}

func (prod *mockProducer) Produce(workers *sync.WaitGroup) {
	// does something.
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
	// Fuck this. distribute for drop route not called! Dont know why :(
	expect := shared.NewExpect(t)
	mockP := mockProducer{
		ProducerBase{
			messages:     make(chan Message),
			control:      make(chan PluginControl),
			streams:      []MessageStreamID{},
			dropStreamID: 2,
			runState:     new(PluginRunState),
			timeout:      500 * time.Millisecond,
			filter:       &mockFilter{},
			format:       &mockFormatter{},
		},
	}
	mockDistribute := func(msg Message) {
		expect.Equal("ProdEanqueueTest", msg.String())
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
	mockP := mockProducer{
		ProducerBase{
			messages:        make(chan Message, 10),
			control:         make(chan PluginControl),
			streams:         []MessageStreamID{},
			dropStreamID:    2,
			runState:        new(PluginRunState),
			timeout:         500 * time.Millisecond,
			filter:          &mockFilter{},
			format:          &mockFormatter{},
			shutdownTimeout: 10 * time.Millisecond,
		},
	}

	mockP.setState(PluginStateActive)

	handleMessage := func(msg Message) {
		expect.Equal("closeMessageChannel", msg.String())
	}

	msgToSend := Message{
		Data:     []byte("closeMessageChannel"),
		StreamID: 1,
	}
	mockP.messages <- msgToSend

	mockP.CloseMessageChannel(handleMessage)

}
