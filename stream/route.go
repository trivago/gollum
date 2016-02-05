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
	"github.com/trivago/tgo"
)

// Route stream plugin
// Configuration example
//
//   - "stream.Route":
//     Enable: true
//     Stream: "data"
//     Routes:
//        - "db1"
//        - "db2"
//        - "data"
//
// Messages will be routed to the streams configured.
// If no route is configured the message is discarded.
//
// Routes defines a 1:n stream remapping.
// Messages are reassigned to all of stream(s) in this list.
// If no route is set messages are forwarded on the incoming stream.
// When routing to multiple streams, the incoming stream has to be listed explicitly to be used.
//
// This stream defines the same fields as stream.Broadcast.
type Route struct {
	core.StreamBase
	routes []streamWithID
}

type streamWithID struct {
	id     core.MessageStreamID
	stream core.Stream
}

func init() {
	core.TypeRegistry.Register(Route{})
}

func newStreamWithID(streamName string) streamWithID {
	streamID := core.GetStreamID(streamName)
	return streamWithID{
		id:     streamID,
		stream: core.StreamRegistry.GetStream(streamID),
	}
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Route) Configure(conf core.PluginConfig) error {
	errors := tgo.NewErrorStack()
	errors.Push(stream.StreamBase.ConfigureStream(conf, stream.Broadcast))

	routes, err := conf.GetStringArray("Routes", []string{})
	errors.Push(err)

	for _, streamName := range routes {
		targetStream := newStreamWithID(streamName)
		stream.routes = append(stream.routes, targetStream)
	}

	return errors.OrNil()
}

func (stream *Route) routeMessage(msg core.Message) {
	for i := 0; i < len(stream.routes); i++ {
		target := stream.routes[i]

		// Stream might require late binding
		if target.stream == nil {
			if core.StreamRegistry.WildcardProducersExist() {
				target.stream = core.StreamRegistry.GetStreamOrFallback(target.id)
			} else if target.stream = core.StreamRegistry.GetStream(target.id); target.stream == nil {
				// Remove without preserving order allows us to continue iterating
				lastIdx := len(stream.routes) - 1
				stream.routes[i] = stream.routes[lastIdx]
				stream.routes = stream.routes[:lastIdx]
				i--
				continue // ### continue, no route ###
			}
		}

		if target.id == stream.GetBoundStreamID() {
			stream.StreamBase.Route(msg, stream.GetBoundStreamID())
		} else {
			msg := msg // copy to allow streamId changes and multiple routes
			msg.StreamID = target.id
			target.stream.Enqueue(msg)
		}
	}
}

// Enqueue overloads the standard Enqueue method to allow direct routing to
// explicit stream targets
func (stream *Route) Enqueue(msg core.Message) {
	if stream.Filter.Accepts(msg) {
		var streamID core.MessageStreamID
		msg.Data, streamID = stream.Format(msg)

		if msg.StreamID != streamID {
			stream.StreamBase.Route(msg, streamID)
			return // ### return, routed by standard method ###
		}

		stream.routeMessage(msg)

		if len(stream.routes) == 0 {
			core.CountNoRouteForMessage()
			return // ### return, no route to producer ###
		}
	}
}
