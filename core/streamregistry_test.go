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
	"github.com/trivago/tgo/ttesting"
	"sync"
	"testing"
)

func getMockStreamRegistry() streamRegistry {
	return streamRegistry{
		routers:     map[MessageStreamID]Router{},
		name:        map[MessageStreamID]string{},
		streamGuard: new(sync.RWMutex),
		nameGuard:   new(sync.RWMutex),
		wildcard:    []Producer{},
	}
}

func TestStreamRegistryGetStreamName(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	expect.Equal(mockSRegistry.GetStreamName(LogInternalStreamID), LogInternalStream)
	expect.Equal(mockSRegistry.GetStreamName(WildcardStreamID), WildcardStream)

	mockStreamID := StreamRegistry.GetStreamID("test")
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "")

	mockSRegistry.name[mockStreamID] = "test"
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "test")
}

func TestStreamRegistryGetStreamByName(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := mockSRegistry.GetRouterByStreamName("testStream")
	expect.Equal(streamName, nil)

	mockStreamID := StreamRegistry.GetStreamID("testStream")
	// TODO: Get a real stream and test with that
	mockSRegistry.routers[mockStreamID] = &mockRouter{}
	expect.Equal(mockSRegistry.GetRouterByStreamName("testStream"), &mockRouter{})
}

func TestStreamRegistryIsStreamRegistered(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	mockStreamID := StreamRegistry.GetStreamID("testStream")

	expect.False(mockSRegistry.IsStreamRegistered(mockStreamID))
	// TODO: Get a real stream and test with that
	mockSRegistry.routers[mockStreamID] = &mockRouter{}
	expect.True(mockSRegistry.IsStreamRegistered(mockStreamID))
}

func TestStreamRegistryForEachStream(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	callback := func(streamID MessageStreamID, stream Router) {
		expect.Equal(streamID, StreamRegistry.GetStreamID("testRegistry"))
	}

	mockSRegistry.routers[StreamRegistry.GetStreamID("testRegistry")] = &mockRouter{}
	mockSRegistry.ForEachStream(callback)
}

func TestStreamRegistryWildcardProducer(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()
	// WildcardProducersExist()
	expect.False(mockSRegistry.WildcardProducersExist())

	producer1 := new(mockBufferedProducer)
	producer2 := new(mockBufferedProducer)

	mockSRegistry.RegisterWildcardProducer(producer1, producer2)

	expect.True(mockSRegistry.WildcardProducersExist())
}

func TestStreamRegistryAddWildcardProducersToStream(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	// create stream to which wildcardProducer is to be added
	mockRouter := getMockRouter()

	// create wildcardProducer.
	mProducer := new(mockBufferedProducer)
	// adding fallbackStreamID to verify the producer later.
	mProducer.fallbackStream = StreamRegistry.GetRouter(StreamRegistry.GetStreamID("wildcardProducerDrop"))
	mockSRegistry.RegisterWildcardProducer(mProducer)

	mockSRegistry.AddWildcardProducersToRouter(&mockRouter)

	streamsProducer := mockRouter.GetProducers()
	expect.Equal(len(streamsProducer), 1)

	// GetDropStreamID removed from  Producer interface in v0.5.0
	// expect.Equal(streamsProducer[0].GetDropStreamID(), StreamRegistry.GetStreamID("wildcardProducerDrop"))

}

func TestStreamRegistryRegister(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := "testStream"
	mockRouter := getMockRouter()
	mockSRegistry.Register(&mockRouter, StreamRegistry.GetStreamID(streamName))

	expect.NotNil(mockSRegistry.GetRouter(StreamRegistry.GetStreamID(streamName)))
}

func TestStreamRegistryGetStreamOrFallback(t *testing.T) {
	// TODO
	// Currently, because StreamRegistry.createFallback() has implicit
	// dependecy on stream.Broadcast we cannot write test case in core
	// package. We should think about alternative way.
}

func TestStreamRegistryConcurrency(t *testing.T) {
	// TODO
	// Currently, because StreamRegistry.createFallback() has implicit
	// dependecy on stream.Broadcast we cannot write test case in core
	// package. We should think about alternative way.
}
