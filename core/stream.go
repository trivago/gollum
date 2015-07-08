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
	"sync/atomic"
	"time"
)

// MessageCount holds the number of messages processed since the last call to
// GetAndResetMessageCount.
var MessageCount = uint32(0)

// Stream defines the interface for all stream plugins
type Stream interface {
	Plugin

	// Pause causes this stream to go silent. Messages should be queued or cause
	// a blocking call. The passed capacity can be used to configure internal
	// channel for buffering incoming messages while this stream is paused.
	Pause(capacity int)

	// Resume causes this stream to send messages again after Pause() had been
	// called. Any buffered messages need to be sent by this method or by a
	// separate go routine.
	Resume()

	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Enqueue sends a given message to all registered producers
	Enqueue(msg Message)
}

// MappedStream holds a stream and the id the stream is assgined to
type MappedStream struct {
	StreamID MessageStreamID
	Stream   Stream
}

// StreamBase defines the standard stream implementation. New stream types
// should derive from this class.
// StreamBase allows streams to set and execute filters as well as format a
// message. Types derived from StreamBase should set the Distribute member
// instead of overloading the Enqueue method.
// See stream.Broadcast for default configuration values and examples.
type StreamBase struct {
	Stream
	Filter            Filter
	Format            Formatter
	Producers         []Producer
	ProducersByStream map[MessageStreamID][]Producer
	StickyStream      bool
	Timeout           *time.Duration
	Distribute        func(msg Message)
	prevDistribute    func(msg Message)
	paused            chan Message
}

// GetAndResetMessageCount returns the current message counter and resets it
// to 0. This function is threadsafe.
func GetAndResetMessageCount() uint32 {
	return atomic.SwapUint32(&MessageCount, 0)
}

// Configure sets up all values requred by StreamBase
func (stream *StreamBase) Configure(conf PluginConfig) error {
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
	stream.ProducersByStream = make(map[MessageStreamID][]Producer)
	stream.StickyStream = conf.GetBool("StickyStream", false)

	if conf.HasValue("Timeout") {
		timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
		stream.Timeout = &timeout
	} else {
		stream.Timeout = nil
	}

	if stream.StickyStream {
		stream.Distribute = stream.broadcastOverStream
	} else {
		stream.Distribute = stream.broadcastOverAll
	}
	return nil
}

// AddProducer adds all producers to the list of known producers.
// Duplicates will be filtered.
func (stream *StreamBase) AddProducer(producers ...Producer) {
	for _, prod := range producers {
		// Fill the "producers by stream" list
	streams:
		for _, streamID := range prod.Streams() {
			for _, inListProd := range stream.ProducersByStream[streamID] {
				if inListProd == prod {
					continue streams // ### continue, already in list ###
				}
			}
			stream.ProducersByStream[streamID] = append(stream.ProducersByStream[streamID], prod)
		}

		// Fill the "all producers" list
		for _, inListProd := range stream.Producers {
			if inListProd == prod {
				return // ### return, already in list ###
			}
		}
		stream.Producers = append(stream.Producers, prod)
	}
}

// Pause will cause this stream to go silent. Messages will be queued to an
// internal channel that can be configured in size by setting the capacity
// parameter. Pass a capacity of 0 to disable buffering.
// Calling Pause on an already paused stream is ignored.
func (stream *StreamBase) Pause(capacity int) {
	if stream.paused == nil {
		stream.paused = make(chan Message, capacity)
		stream.prevDistribute = stream.Distribute
		stream.Distribute = stream.stash
	}
}

// Resume causes this stream to send messages again after Pause() had been
// called. Any buffered messages will be sent by a separate go routine.
// Calling Resume on a stream that is not paused is ignored.
func (stream *StreamBase) Resume() {
	if stream.paused != nil {
		stream.Distribute = stream.prevDistribute

		stashed := stream.paused
		stream.paused = nil
		close(stashed)

		go func() {
			for msg := range stashed {
				stream.Distribute(msg)
			}
		}()
	}
}

func (stream *StreamBase) stash(msg Message) {
	stream.paused <- msg
}

func (stream *StreamBase) broadcastOverAll(msg Message) {
	for _, prod := range stream.Producers {
		prod.Enqueue(msg, stream.Timeout)
	}
}

func (stream *StreamBase) broadcastOverStream(msg Message) {
	producers, exists := stream.ProducersByStream[msg.StreamID]
	if !exists {
		shared.Metric.Inc(MetricNoRoute)
		shared.Metric.Inc(MetricDiscarded)
	}
	for _, prod := range producers {
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
	}
}

// Route is called by Enqueue after a message has been accepted and formatted.
// This encapsulates the main logic of sending messages to producers or to
// another stream if necessary.
func (stream *StreamBase) Route(msg Message, targetID MessageStreamID) {
	if msg.StreamID != targetID {
		msg.StreamID = targetID
		StreamTypes.GetStreamOrFallback(targetID).Enqueue(msg)
		return // ### done, routed ###
	}

	if len(stream.Producers) == 0 {
		shared.Metric.Inc(MetricNoRoute)
		shared.Metric.Inc(MetricDiscarded)
		return // ### return, no route to producer ###
	}

	atomic.AddUint32(&MessageCount, 1)
	stream.Distribute(msg)
}
