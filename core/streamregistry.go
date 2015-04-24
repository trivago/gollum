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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"hash/fnv"
)

const (
	metricStreams = "Streams"
)

// StreamRegistry holds streams mapped by their MessageStreamID as well as a
// reverse lookup of MessageStreamID to stream name.
type StreamRegistry struct {
	streams  map[MessageStreamID]Stream
	name     map[MessageStreamID]string
	wildcard []Producer
}

// StreamTypes is the global instance of StreamRegistry used to store the
// all registered streams.
var StreamTypes = StreamRegistry{
	streams: make(map[MessageStreamID]Stream),
	name:    make(map[MessageStreamID]string),
}

func init() {
	shared.Metric.New(metricStreams)
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	streamID := MessageStreamID(hash.Sum64())

	StreamTypes.name[streamID] = stream
	return streamID
}

// GetStreamName does a reverse lookup for a given MessageStreamID and returns
// the corresponding name. If the MessageStreamID is not registered, an empty
// string is returned.
func (registry StreamRegistry) GetStreamName(streamID MessageStreamID) string {
	if name, exists := registry.name[streamID]; exists {
		return name // ### return, found ###
	}
	return ""
}

// GetStreamByName returns a registered stream by name. See GetStream.
func (registry StreamRegistry) GetStreamByName(name string) Stream {
	streamID := GetStreamID(name)
	return registry.GetStream(streamID)
}

// GetStream returns a registered stream or nil
func (registry StreamRegistry) GetStream(id MessageStreamID) Stream {
	stream, exists := registry.streams[id]
	if !exists {
		return nil
	}
	return stream
}

// IsStreamRegistered returns true if the stream for the given id is registered.
func (registry StreamRegistry) IsStreamRegistered(id MessageStreamID) bool {
	_, exists := registry.streams[id]
	return exists
}

// ForEachStream loops over all registered streams and calls the given function.
func (registry StreamRegistry) ForEachStream(callback func(streamID MessageStreamID, stream Stream)) {
	for streamID, stream := range registry.streams {
		callback(streamID, stream)
	}
}

// RegisterWildcardProducer adds a new producer to the list of known wildcard
// prodcuers. This list has to be added to new streams upon creation to send
// messages to producers listening to *.
// Duplicates will be filtered.
// This state of this list is undefined during the configuration phase.
func (registry *StreamRegistry) RegisterWildcardProducer(producers ...Producer) {
nextProd:
	for _, prod := range producers {
		for _, existing := range registry.wildcard {
			if existing == prod {
				continue nextProd
			}
		}
		registry.wildcard = append(registry.wildcard, prod)
	}
}

// AddWildcardProducersToStream adds all registered wildcard producers to a
// given stream.
func (registry StreamRegistry) AddWildcardProducersToStream(stream Stream) {
	stream.AddProducer(registry.wildcard...)
}

// Register registeres a stream plugin to a given stream id
func (registry *StreamRegistry) Register(stream Stream, streamID MessageStreamID) {
	if _, exists := registry.streams[streamID]; exists {
		Log.Warning.Printf("%T attaches to an already occupied stream (%s)", stream, registry.GetStreamName(streamID))
	} else {
		shared.Metric.Inc(metricStreams)
	}
	registry.streams[streamID] = stream
}

// GetStreamOrFallback returns the stream for the given id if it is registered.
// If no stream is registered for the given id the default stream is used.
// The default stream is equivalent to an unconfigured stream.Broadcast with
// all wildcard producers allready added.
func (registry *StreamRegistry) GetStreamOrFallback(streamID MessageStreamID) Stream {
	if stream, exists := registry.streams[streamID]; exists {
		return stream
	}

	defaultStream := new(StreamBase)
	defaultStream.Configure(PluginConfig{})
	registry.AddWildcardProducersToStream(defaultStream)

	registry.streams[streamID] = defaultStream
	shared.Metric.Inc(metricStreams)
	return defaultStream
}
