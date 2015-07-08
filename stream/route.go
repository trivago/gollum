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

package stream

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Route stream plugin
// Configuration example
//
//   - "stream.Route":
//     Enable: true
//     Stream:
//       - "data"
//       - "_DROPPED_"
//     Routes:
//        "_DROPPED_": "myStream"
//        "data":
//			- "db1"
//          - "db2"
//
// Messages will be routed to the streams configured.
// If no route is configured the message is discarded.
//
// Routes defines a 1:n stream remapping. Messages reaching the Route stream
// are reassigned to the given stream(s). If no Route is set the message will
// be broadcasted to all producers attached to this stream.
//
// This stream defines the same fields as stream.Broadcast.
type Route struct {
	core.StreamBase
	routes map[core.MessageStreamID][]streamWithID
}

type streamWithID struct {
	id     core.MessageStreamID
	stream core.Stream
}

func init() {
	shared.RuntimeType.Register(Route{})
}

func newStreamWithID(streamID core.MessageStreamID) streamWithID {
	return streamWithID{
		id:     streamID,
		stream: core.StreamTypes.GetStream(streamID),
	}
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Route) Configure(conf core.PluginConfig) error {
	if err := stream.StreamBase.Configure(conf); err != nil {
		return err // ### return, base stream error ###
	}

	routes := conf.GetStreamRoutes("Routes")
	stream.routes = make(map[core.MessageStreamID][]streamWithID)

	for sourceID, targetIDs := range routes {
		for _, targetID := range targetIDs {
			stream.addRoute(sourceID, targetID)
		}
	}

	return nil
}

func (stream *Route) addRoute(sourceID core.MessageStreamID, targetID core.MessageStreamID) {
	targetStream := newStreamWithID(targetID)

	if _, exists := stream.routes[sourceID]; !exists {
		stream.routes[sourceID] = []streamWithID{targetStream}
	} else {
		stream.routes[sourceID] = append(stream.routes[sourceID], targetStream)
	}
}

func (stream *Route) removeRoute(sourceID core.MessageStreamID, targetID core.MessageStreamID) {
	streams := stream.routes[sourceID]
	if len(streams) == 1 {
		delete(stream.routes, sourceID)
		return // ### return, last entry ###
	}

	for idx, target := range streams {
		if target.id == targetID {
			switch {
			case idx == 0:
				stream.routes[sourceID] = streams[1:]
			case idx == len(streams)-1:
				stream.routes[sourceID] = streams[:idx]
			default:
				stream.routes[sourceID] = append(streams[:idx], streams[idx+1:]...)
			}
			return // ### return, removed entry ###
		}
	}
}

// Enqueue routes the given message to another stream.
func (stream *Route) Enqueue(msg core.Message) {
	// Do standard filtering and formatting
	if stream.Filter.Accepts(msg) {
		var streamID core.MessageStreamID
		msg.Data, streamID = stream.Format.Format(msg)
		routeSuccess := false

		// Search for route targets and enqueue
		if streams, routeExists := stream.routes[streamID]; routeExists {
			for _, target := range streams {
				// Stream might require late binding
				if target.stream == nil {
					if target.stream = core.StreamTypes.GetStream(target.id); target.stream == nil {
						defer stream.removeRoute(streamID, target.id)
						continue // ### continue, no route ###
					}
				}

				if target.id == msg.StreamID {
					stream.Route(msg, streamID) // Route to self
				} else {
					msg := msg // copy to allow streamId changes and multiple routes
					msg.StreamID = target.id
					target.stream.Enqueue(msg)
				}
				routeSuccess = true
			}
		}

		// If message could not be routed, try direct send
		if !routeSuccess {
			stream.Route(msg, streamID)
		}
	}
}
