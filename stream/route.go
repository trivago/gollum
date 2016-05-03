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

package stream

import (
	"github.com/trivago/gollum/core"
)

// Route stream plugin
// Messages will be routed to all streams configured. Each target stream can
// hold another stream configuration, too, so this is not directly sending to
// the producers attached to the target streams.
// Configuration example
//
//  - "stream.Route":
//    Routes:
//      - "foo"
//      - "bar"
//
// Routes defines a 1:n stream remapping.
// Messages are reassigned to all of stream(s) in this list.
// If no route is set messages are forwarded on the incoming stream.
// When routing to multiple streams, the incoming stream has to be listed explicitly to be used.
type Route struct {
	Broadcast
	streams []core.Stream
}

func init() {
	core.TypeRegistry.Register(Route{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Route) Configure(conf core.PluginConfigReader) error {
	stream.Broadcast.Configure(conf)

	boundStreamIDs := conf.GetStreamArray("Streams", []core.MessageStreamID{})
	for _, streamID := range boundStreamIDs {
		route := core.StreamRegistry.GetStreamOrFallback(streamID)
		stream.streams = append(stream.streams, route)
	}

	return conf.Errors.OrNil()
}

// Enqueue overloads the standard Enqueue method to allow direct routing to
// explicit stream targets
func (stream *Route) Enqueue(msg *core.Message) bool {
	if len(stream.streams) == 0 {
		return false // ### return, no route ###
	}

	if len(stream.streams) == 1 {
		route := stream.streams[0]
		if route.StreamID() == stream.StreamID() {
			return stream.Broadcast.Enqueue(msg) // ### return, broadcast ###
		}

		msg.SetStreamID(route.StreamID())
		core.Route(msg, route)
		return true // ### return, fast path ###
	}

	for _, route := range stream.streams {
		if route.StreamID() == stream.StreamID() {
			stream.Broadcast.Enqueue(msg)
		} else {
			msg := msg.Clone()
			msg.SetStreamID(route.StreamID())
			core.Route(msg, route)
		}
	}

	return true
}
