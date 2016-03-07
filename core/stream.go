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
	"sync"
	"time"
)

// Stream defines the interface for all stream plugins
type Stream interface {
	// GetBoundStreamID returns the stream id this plugin is bound to.
	GetBoundStreamID() MessageStreamID

	// Pause causes this stream to go silent. Messages should be queued or cause
	// a blocking call. The passed capacity can be used to configure internal
	// channel for buffering incoming messages while this stream is paused.
	Pause(capacity int)

	// Resume causes this stream to send messages again after Pause() had been
	// called. Any buffered messages need to be sent by this method or by a
	// separate go routine.
	Resume()

	// Flush calls Resume and blocks until resume finishes
	Flush()

	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Enqueue sends a given message to all registered producers
	Enqueue(msg Message)

	// GetProducers returns the producers bound to this stream
	GetProducers() []Producer
}

// MappedStream holds a stream and the id the stream is assgined to
type MappedStream struct {
	StreamID MessageStreamID
	Stream   Stream
}

// StreamBase plugin base type
// This type defines the standard stream implementation. New stream types
// should derive from this class.
// StreamBase allows streams to set and execute filters as well as format a
// message. Types derived from StreamBase should set the Distribute member
// instead of overloading the Enqueue method.
// Configuration Example
//
//  - "stream.Foobar"
//    Enable: true
//    Stream: "streamToConfigure"
//    Formatter: "format.Forward"
//    Filter: "filter.All"
//    TimeoutMs: 0
//
// Enable can be set to false to disable this stream configuration but leave
// it in the config for future use. Set to true by default.
//
// Stream defines the stream to configure. This is a mandatory setting and
// has no default value.
//
// Formatter defines the first formatter to apply to the messages passing through
// this stream. By default this is set to "format.Forward".
//
// Filter defines the filter to apply to the messages passing through this stream.
// By default this is et to "filter.All".
//
// TimeoutMs defines an optional timeout that can be used to wait for producers
// attached to this stream to unblock. This setting overwrites the corresponding
// producer setting for this (and only this) stream.
type StreamBase struct {
	Filter         Filter
	Format         Formatter
	Producers      []Producer
	Timeout        *time.Duration
	boundStreamID  MessageStreamID
	distribute     Distributor
	prevDistribute Distributor
	paused         chan Message
	resumeWorker   *sync.WaitGroup
}

// Distributor is a callback typedef for methods processing messages
type Distributor func(msg Message)

// ConfigureStream sets up all values required by StreamBase.
func (stream *StreamBase) ConfigureStream(conf PluginConfig, distribute Distributor) error {
	plugin, err := NewPluginWithType(conf.GetString("Formatter", "format.Forward"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}
	stream.Format = plugin.(Formatter)

	plugin, err = NewPluginWithType(conf.GetString("Filter", "filter.All"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}

	stream.Filter = plugin.(Filter)
	if len(conf.Stream) == 0 {
		panic("No source stream configured.")
	}

	stream.boundStreamID = StreamRegistry.GetStreamID(conf.Stream[0])
	stream.resumeWorker = new(sync.WaitGroup)
	stream.distribute = distribute

	if conf.HasValue("TimeoutMs") {
		timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
		stream.Timeout = &timeout
	} else {
		stream.Timeout = nil
	}
	return nil
}

// AddProducer adds all producers to the list of known producers.
// Duplicates will be filtered.
func (stream *StreamBase) AddProducer(producers ...Producer) {
	for _, prod := range producers {
		for _, inListProd := range stream.Producers {
			if inListProd == prod {
				return // ### return, already in list ###
			}
		}
		stream.Producers = append(stream.Producers, prod)
	}
}

// GetProducers returns the producers bound to this stream
func (stream *StreamBase) GetProducers() []Producer {
	return stream.Producers
}

// Pause will cause this stream to go silent. Messages will be queued to an
// internal channel that can be configured in size by setting the capacity
// parameter. Pass a capacity of 0 to disable buffering.
// Calling Pause on an already paused stream is ignored.
func (stream *StreamBase) Pause(capacity int) {
	if stream.paused == nil {
		stream.paused = make(chan Message, capacity)
		stream.prevDistribute = stream.distribute
		stream.distribute = stream.stash
	}
}

// GetBoundStreamID returns the id of the stream this plugin is bound to.
func (stream *StreamBase) GetBoundStreamID() MessageStreamID {
	return stream.boundStreamID
}

// Resume causes this stream to send messages again after Pause() had been
// called. Any buffered messages will be sent by a separate go routine.
// Calling Resume on a stream that is not paused is ignored.
func (stream *StreamBase) Resume() {
	if stream.paused != nil {
		stream.distribute = stream.prevDistribute
		stream.resumeWorker.Add(1)

		stashed := stream.paused
		stream.paused = nil
		close(stashed)

		go func() {
			for msg := range stashed {
				stream.distribute(msg)
			}
			stream.resumeWorker.Done()
		}()
	}
}

// Flush calls Resume and blocks until resume finishes
func (stream *StreamBase) Flush() {
	stream.Resume()
	stream.resumeWorker.Wait()
}

// stash is used as a distributor during pause
func (stream *StreamBase) stash(msg Message) {
	stream.paused <- msg
}

// Broadcast enqueues the given message to all producers attached to this stream.
func (stream *StreamBase) Broadcast(msg Message) {
	for _, prod := range stream.Producers {
		prod.Enqueue(msg, stream.Timeout)
	}
}

// Enqueue checks the filter, formats the message and sends it to all producers
// registered. Functions deriving from StreamBase can set the Distribute member
// to hook into this function.
func (stream *StreamBase) Enqueue(msg Message) {
	if stream.Filter.Accepts(msg) {
		var streamID MessageStreamID
		msg.Data, streamID = stream.Format.Format(msg)
		stream.Route(msg, streamID)
	} else {
		CountFilteredMessage()
	}
}

// Route is called by Enqueue after a message has been accepted and formatted.
// This encapsulates the main logic of sending messages to producers or to
// another stream if necessary.
func (stream *StreamBase) Route(msg Message, targetID MessageStreamID) {
	if msg.StreamID != targetID {
		msg.PrevStreamID = msg.StreamID
		msg.StreamID = targetID
		StreamRegistry.GetStreamOrFallback(targetID).Enqueue(msg)
		return // ### done, routed ###
	}

	if len(stream.Producers) == 0 {
		CountNoRouteForMessage()
		//Log.Debug.Print("No producers for ", StreamRegistry.GetStreamName(msg.StreamID))
		return // ### return, no route to producer ###
	}

	CountProcessedMessage()
	stream.distribute(msg)
}
