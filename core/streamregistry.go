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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tsync"
	"hash/fnv"
	"sync"
	"time"
)

const (
	metricStreams  = "Streams"
	metricMessages = "Messages"
	// MetricMessagesSec is used as a key for storing message throughput
	MetricMessagesSec  = "MessagesPerSec"
	metricDiscarded    = "DiscardedMessages"
	metricDropped      = "DroppedMessages"
	metricFiltered     = "Filtered"
	metricNoRoute      = "DiscardedNoRoute"
	metricDiscardedSec = "DiscardedMessagesSec"
	metricDroppedSec   = "DroppedMessagesSec"
	metricFilteredSec  = "FilteredSec"
	metricNoRouteSec   = "DiscardedNoRouteSec"
)

// streamRegistry holds streams mapped by their MessageStreamID as well as a
// reverse lookup of MessageStreamID to stream name.
type streamRegistry struct {
	streams     map[MessageStreamID]Stream
	name        map[MessageStreamID]string
	fuses       map[string]*tsync.Fuse
	fuseGuard   *sync.Mutex
	nameGuard   *sync.Mutex
	streamGuard *sync.Mutex
	wildcard    []Producer
}

// StreamRegistry is the global instance of streamRegistry used to store the
// all registered streams.
var StreamRegistry = streamRegistry{
	streams:     make(map[MessageStreamID]Stream),
	streamGuard: new(sync.Mutex),
	name:        make(map[MessageStreamID]string),
	nameGuard:   new(sync.Mutex),
	fuses:       make(map[string]*tsync.Fuse),
	fuseGuard:   new(sync.Mutex),
}

func init() {
	tgo.EnableGlobalMetrics()
	tgo.Metric.New(metricStreams)
	tgo.Metric.New(metricMessages)
	tgo.Metric.New(metricDropped)
	tgo.Metric.New(metricDiscarded)
	tgo.Metric.New(metricNoRoute)
	tgo.Metric.New(metricFiltered)
	tgo.Metric.NewRate(metricDropped, metricDroppedSec, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricMessages, MetricMessagesSec, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricDiscarded, metricDiscardedSec, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricNoRoute, metricNoRouteSec, time.Second, 10, 3, true)
	tgo.Metric.NewRate(metricFiltered, metricFilteredSec, time.Second, 10, 3, true)
}

// CountProcessedMessage increases the messages counter by 1
func CountProcessedMessage() {
	tgo.Metric.Inc(metricMessages)
}

// CountDroppedMessage increases the dropped messages counter by 1
func CountDroppedMessage() {
	tgo.Metric.Inc(metricDropped)
}

// CountDiscardedMessage increases the discarded messages counter by 1
func CountDiscardedMessage() {
	tgo.Metric.Inc(metricDiscarded)
}

// CountFilteredMessage increases the filtered messages counter by 1
func CountFilteredMessage() {
	tgo.Metric.Inc(metricFiltered)
}

// CountNoRouteForMessage increases the "no route" counter by 1
func CountNoRouteForMessage() {
	tgo.Metric.Inc(metricNoRoute)
}

// GetStreamID is deprecated
func GetStreamID(stream string) MessageStreamID {
	return StreamRegistry.GetStreamID(stream)
}

// GetStreamID returns the integer representation of a given stream name.
func (registry *streamRegistry) GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	streamID := MessageStreamID(hash.Sum64())

	registry.nameGuard.Lock()
	defer registry.nameGuard.Unlock()
	registry.name[streamID] = stream

	return streamID
}

// GetStreamName does a reverse lookup for a given MessageStreamID and returns
// the corresponding name. If the MessageStreamID is not registered, an empty
// string is returned.
func (registry streamRegistry) GetStreamName(streamID MessageStreamID) string {
	switch streamID {
	case DroppedStreamID:
		return DroppedStream

	case LogInternalStreamID:
		return LogInternalStream

	case WildcardStreamID:
		return WildcardStream

	default:
		registry.nameGuard.Lock()
		defer registry.nameGuard.Unlock()
		if name, exists := registry.name[streamID]; exists {
			return name // ### return, found ###
		}
	}
	return ""
}

// GetStreamByName returns a registered stream by name. See GetStream.
func (registry streamRegistry) GetStreamByName(name string) Stream {
	streamID := registry.GetStreamID(name)
	return registry.GetStream(streamID)
}

// GetStream returns a registered stream or nil
func (registry streamRegistry) GetStream(id MessageStreamID) Stream {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()
	stream, exists := registry.streams[id]
	if !exists {
		return nil
	}
	return stream
}

// IsStreamRegistered returns true if the stream for the given id is registered.
func (registry streamRegistry) IsStreamRegistered(id MessageStreamID) bool {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()
	_, exists := registry.streams[id]
	return exists
}

// ForEachStream loops over all registered streams and calls the given function.
func (registry streamRegistry) ForEachStream(callback func(streamID MessageStreamID, stream Stream)) {
	registry.streamGuard.Lock()
	streams := registry.streams
	registry.streamGuard.Unlock()

	for streamID, stream := range streams {
		callback(streamID, stream)
	}
}

// WildcardProducersExist returns true if any producer is listening to the
// wildcard stream.
func (registry *streamRegistry) WildcardProducersExist() bool {
	return len(registry.wildcard) > 0
}

// RegisterWildcardProducer adds a new producer to the list of known wildcard
// prodcuers. This list has to be added to new streams upon creation to send
// messages to producers listening to *.
// Duplicates will be filtered.
// This state of this list is undefined during the configuration phase.
func (registry *streamRegistry) RegisterWildcardProducer(producers ...Producer) {
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

// AddWildcardProducersToStream adds all known wildcard producers to a given
// stream. The state of the wildcard list is undefined during the configuration
// phase.
func (registry streamRegistry) AddWildcardProducersToStream(stream Stream) {
	streamID := stream.GetBoundStreamID()
	if streamID != LogInternalStreamID && streamID != DroppedStreamID {
		stream.AddProducer(registry.wildcard...)
	}
}

// AddAllWildcardProducersToAllStreams executes AddWildcardProducersToStream on
// all currently registered streams
func (registry *streamRegistry) AddAllWildcardProducersToAllStreams() {
	registry.ForEachStream(
		func(streamID MessageStreamID, stream Stream) {
			registry.AddWildcardProducersToStream(stream)
		})
}

// Register registeres a stream plugin to a given stream id
func (registry *streamRegistry) Register(stream Stream, streamID MessageStreamID) {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()

	if _, exists := registry.streams[streamID]; exists {
		tlog.Warning.Printf("%T attaches to an already occupied stream (%s)", stream, registry.GetStreamName(streamID))
	} else {
		tgo.Metric.Inc(metricStreams)
	}
	registry.streams[streamID] = stream
}

// GetStreamOrFallback returns the stream for the given id if it is registered.
// If no stream is registered for the given id the default stream is used.
// The default stream is equivalent to an unconfigured stream.Broadcast with
// all wildcard producers already added.
func (registry *streamRegistry) GetStreamOrFallback(streamID MessageStreamID) Stream {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()
	if stream, exists := registry.streams[streamID]; exists {
		return stream
	}

	streamName := registry.GetStreamName(streamID)
	tlog.Debug.Print("Using fallback stream for ", streamName)

	defaultStream := new(StreamBase)
	defaultConfig := NewPluginConfig("", "core.StreamBase")
	defaultConfig.Override("stream", streamName)

	defaultStream.ConfigureStream(NewPluginConfigReader(&defaultConfig), defaultStream.Broadcast)
	registry.AddWildcardProducersToStream(defaultStream)

	registry.streams[streamID] = defaultStream
	tgo.Metric.Inc(metricStreams)
	return defaultStream
}

// GetFuse returns a fuse object by name. This function will always return a
// valid fuse (creates fuses if they have not yet been created).
// This function is threadsafe.
func (registry *streamRegistry) GetFuse(name string) *tsync.Fuse {
	registry.fuseGuard.Lock()
	defer registry.fuseGuard.Unlock()

	fuse, exists := registry.fuses[name]
	if !exists {
		fuse = tsync.NewFuse()
		registry.fuses[name] = fuse
	}
	return fuse
}

// ActivateAllFuses calls Activate on all registered fuses.
func (registry *streamRegistry) ActivateAllFuses() {
	registry.fuseGuard.Lock()
	defer registry.fuseGuard.Unlock()

	for _, fuse := range registry.fuses {
		fuse.Activate()
	}
}
