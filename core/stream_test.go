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
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"sync"
	"testing"
	"time"
)

func getMockStream() StreamBase {
	timeout := time.Second
	return StreamBase{
		filters:        []Filter{&mockFilter{}},
		formatters:     []Formatter{&mockFormatter{}},
		Timeout:        &timeout,
		Producers:      []Producer{},
		boundStreamID:  GetStreamID("testBoundStream"),
		distribute:     mockDistributer,
		prevDistribute: mockPrevDistributer,
		paused:         make(chan Message),
		resumeWorker:   new(sync.WaitGroup),
	}
}

func registerMockStream(streamName string) {
	TypeRegistry.Register(mockPlugin{})
	TypeRegistry.Register(mockFormatter{})
	TypeRegistry.Register(mockFilter{})

	mockStream := getMockStream()
	mockConf := NewPluginConfig("", "core.mockPlugin")
	mockConf.Override("Stream", streamName)
	mockConf.Override("Formatters", []interface{}{map[string]tcontainer.MarshalMap{"core.mockFormatter": tcontainer.NewMarshalMap()}})
	mockConf.Override("Filter", "core.mockFilter")

	mockStream.ConfigureStream(NewPluginConfigReader(&mockConf), func(msg Message) {})
	StreamRegistry.Register(&mockStream, mockStream.GetBoundStreamID())
}

func TestStreamConfigureStream(t *testing.T) {
	expect := ttesting.NewExpect(t)
	TypeRegistry.Register(mockPlugin{})
	TypeRegistry.Register(mockFormatter{})
	TypeRegistry.Register(mockFilter{})

	mockConf := NewPluginConfig("", "core.mockPlugin")
	mockConf.Override("Stream", "testBoundStream")
	mockConf.Override("Formatters", []interface{}{map[string]tcontainer.MarshalMap{"core.mockFormatter": tcontainer.NewMarshalMap()}})
	mockConf.Override("Filter", "core.mockFilter")
	mockConf.Override("TimeoutMs", 100)

	mockDistributer := func(msg Message) {
		expect.True(true)
	}

	mockStream := getMockStream()
	mockStream.ConfigureStream(NewPluginConfigReader(&mockConf), mockDistributer)
}

func TestStreamPauseFlush(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockStream := getMockStream()

	mockDistributer := func(msg Message) {
		expect.Equal("abc", msg.String())
	}
	// rewrite paused as nil to check if properly assigned by Pause(capacity).
	mockStream.paused = nil
	mockStream.distribute = mockDistributer

	// shouldn't the enqued message after pause start distributing after resume?
	msgToSend := Message{
		Data:     []byte("abc"),
		StreamID: 1,
	}
	mockStream.AddProducer(&mockProducer{})
	mockStream.Pause(1)
	mockStream.Enqueue(msgToSend)
	mockStream.Flush()
}

func TestStreamBroadcast(t *testing.T) {
	// TODO: complete after mockProducer

}

func TestStreamRoute(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockStream := getMockStream()

	mockDistributer := func(msg Message) {
		expect.Equal("abc", msg.String())
	}
	targetMockStream := getMockStream()
	targetMockStream.AddProducer(&mockProducer{})
	targetMockStream.distribute = mockDistributer
	StreamRegistry.streams[2] = &targetMockStream

	msgToSend := Message{
		Data:     []byte("abc"),
		StreamID: 1,
	}

	mockStream.AddProducer(&mockProducer{})
	mockStream.Route(msgToSend, 2)

}
