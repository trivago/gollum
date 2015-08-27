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
	"testing"
)

func getMockStreamRegistry() streamRegistry {
	return streamRegistry{map[MessageStreamID]Stream{},
		map[MessageStreamID]string{},
		[]Producer{},
	}
}

func mockDistributer(msg Message) {

}

func mockPrevDistributer(msg Message) {

}

func resetStreamRegistryCounts() {
	messageCount = 0
	droppedCount = 0
	discardedCount = 0
	filteredCount = 0
	noRouteCount = 0
}

func TestStreamRegistryAtomicOperations(t *testing.T) {
	expect := shared.NewExpect(t)
	// Tests run in one go, so the counts might already be affected. Reset here.
	resetStreamRegistryCounts()

	expect.Equal(messageCount, uint32(0))
	CountProcessedMessage()
	expect.Equal(messageCount, uint32(1))

	expect.Equal(droppedCount, uint32(0))
	CountDroppedMessage()
	expect.Equal(droppedCount, uint32(1))

	expect.Equal(discardedCount, uint32(0))
	CountDiscardedMessage()
	expect.Equal(discardedCount, uint32(1))

	expect.Equal(filteredCount, uint32(0))
	CountFilteredMessage()
	expect.Equal(filteredCount, uint32(1))

	expect.Equal(noRouteCount, uint32(0))
	CountNoRouteForMessage()
	expect.Equal(noRouteCount, uint32(1))

}

func TestStreamRegistryGetStreamName(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	expect.Equal(mockSRegistry.GetStreamName(DroppedStreamID), DroppedStream)
	expect.Equal(mockSRegistry.GetStreamName(LogInternalStreamID), LogInternalStream)
	expect.Equal(mockSRegistry.GetStreamName(WildcardStreamID), WildcardStream)

	mockStreamID := GetStreamID("test")
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "")

	mockSRegistry.name[mockStreamID] = "test"
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "test")
}

func TestStreamRegistryGetStreamByName(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := mockSRegistry.GetStreamByName("testStream")
	expect.Equal(streamName, nil)

	mockStreamID := GetStreamID("testStream")
	// TODO: Get a real stream and test with that
	mockSRegistry.streams[mockStreamID] = &StreamBase{}
	expect.Equal(mockSRegistry.GetStreamByName("testStream"), &StreamBase{})
}

func TestStreamRegistryIsStreamRegistered(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	mockStreamID := GetStreamID("testStream")

	expect.False(mockSRegistry.IsStreamRegistered(mockStreamID))
	// TODO: Get a real stream and test with that
	mockSRegistry.streams[mockStreamID] = &StreamBase{}
	expect.True(mockSRegistry.IsStreamRegistered(mockStreamID))
}

func TestStreamRegistryForEachStream(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	callback := func(streamID MessageStreamID, stream Stream) {
		expect.Equal(streamID, GetStreamID("testRegistry"))
	}

	mockSRegistry.streams[GetStreamID("testRegistry")] = &StreamBase{}
	mockSRegistry.ForEachStream(callback)
}

func TestStreamRegistryWildcardProducer(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()
	// WildcardProducersExist()
	expect.False(mockSRegistry.WildcardProducersExist())

	producer1 := new(mockProducer)
	producer2 := new(mockProducer)

	mockSRegistry.RegisterWildcardProducer(producer1, producer2)

	expect.True(mockSRegistry.WildcardProducersExist())
}

func TestStreamRegistryAddWildcardProducersToStream(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	// create stream to which wildcardProducer is to be added
	mockStream := getMockStream()

	// create wildcardProducer.
	mProducer := new(mockProducer)
	// adding dropStreamID to verify the producer later.
	mProducer.dropStreamID = GetStreamID("wildcardProducerDrop")
	mockSRegistry.RegisterWildcardProducer(mProducer)

	mockSRegistry.AddWildcardProducersToStream(&mockStream)

	streamsProducer := mockStream.GetProducers()
	expect.Equal(len(streamsProducer), 1)

	expect.Equal(streamsProducer[0].GetDropStreamID(), GetStreamID("wildcardProducerDrop"))
}

func TestStreamRegistryRegister(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := "testStream"
	mockStream := getMockStream()
	mockSRegistry.Register(&mockStream, GetStreamID(streamName))

	expect.NotNil(mockSRegistry.GetStream(GetStreamID(streamName)))
}

func TestStreamRegistryGetStreamOrFallback(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	expect.Equal(len(mockSRegistry.streams), 0)
	expect.Equal(len(mockSRegistry.wildcard), 0)

	streamName := "testStream"
	streamID := GetStreamID(streamName)
	mockSRegistry.GetStreamOrFallback(streamID)

	expect.Equal(len(mockSRegistry.streams), 1)

	// try registering again. No new register should happen.
	mockSRegistry.GetStreamOrFallback(streamID)
	expect.Equal(len(mockSRegistry.streams), 1)
}

func TestStreamRegistryLinkDependencies(t *testing.T) {
	/**
	The dependency tree in this test is given below. A -> B means A depends on B
		producer1 -> producer3
		producer3 -> producer2
		producer2 -> none
		producer4 -> none
	*/
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := "testStream"
	streamID := GetStreamID(streamName)
	//register a stream
	stream := mockSRegistry.GetStreamOrFallback(streamID)

	producer1 := mockProducer{}
	producer2 := mockProducer{}
	producer3 := mockProducer{}
	producer4 := mockProducer{}

	expect.Equal(len(producer1.dependencies), 0)
	expect.Equal(len(producer2.dependencies), 0)
	expect.Equal(len(producer3.dependencies), 0)
	expect.Equal(len(producer4.dependencies), 0)

	producer1.AddDependency(&producer3)
	producer3.AddDependency(&producer2)

	stream.AddProducer(&producer1, &producer2, &producer3, &producer4)

	mockSRegistry.LinkDependencies(&producer3, streamID)

	expect.Equal(len(producer1.dependencies), 1)
	expect.Equal(len(producer2.dependencies), 0)
	expect.Equal(len(producer3.dependencies), 1)
	expect.Equal(len(producer4.dependencies), 1)

}
