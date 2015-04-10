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
	"sync"
	"sync/atomic"
)

// MessageCount holds the number of messages processed since the last call to
// GetAndResetMessageCount.
var MessageCount = uint32(0)

// Stream defines the interface for all stream plugins
type Stream interface {
	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Enqueue sends a given message to all registered producers
	Enqueue(msg Message)
}

// StreamBase defines the standard stream implementation. New stream types
// should derive from this class.
// StreamBase allows streams to set and execute filters as well as format a
// message. Types derived from StreamBase should set the Distribute member
// instead of overloading the Enqueue method.
// See stream.Broadcast for default configuration values and examples.
type StreamBase struct {
	Filter     Filter
	Format     Formatter
	Producers  []Producer
	Distribute func(msg Message)
	queueLock  *sync.Mutex
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

	stream.Distribute = stream.broadcast
	stream.queueLock = new(sync.Mutex)
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

func (stream *StreamBase) broadcast(msg Message) {
	for _, prod := range stream.Producers {
		prod.Enqueue(msg)
	}
}

// Enqueue checks the filter, formats the message and sends it to all producers
// registered. Functions deriving from StreamBase can set the Distribute member
// to hook into this function.
func (stream *StreamBase) Enqueue(msg Message) {
	atomic.AddUint32(&MessageCount, 1)

	// As filters and/or formatter may store internal states we have to make
	// sure that these are not modified by parallel calls to this method.
	stream.queueLock.Lock()
	defer stream.queueLock.Unlock()

	if stream.Filter.Accepts(msg) {
		stream.Format.PrepareMessage(msg)
		msg.Data = stream.Format.Bytes()
		stream.Distribute(msg)
	}
}
