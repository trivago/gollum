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

// Stream defines the interface for all stream plugins
type Stream interface {
	// StreamID returns the stream id this plugin is bound to.
	StreamID() MessageStreamID

	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Format calls all formatters in their order of definition
	Format(msg *Message)

	// Accepts calls applys all filters to the given message and returns true when
	// all filters pass.
	Accepts(msg *Message) bool

	// Enqueue sends a given message to all registered end points.
	// This function is called by Route() which should be preferred over this
	// function when sending messages to a stream unless you don't care about
	// formatters that can change the stream of a message.
	// If the message could be enqueued true is returned
	Enqueue(msg *Message) bool
}

// Route tries to enqueue a message to the given stream. This function also
// handles redirections enforced by formatters.
func Route(msg *Message, stream Stream) {
	if stream.Accepts(msg) {
		stream.Format(msg)

		if msg.StreamID() != stream.StreamID() {
			detour := StreamRegistry.GetStreamOrFallback(msg.StreamID())
			Route(msg, detour)
			return // ### return, detour ###
		}

		if stream.Enqueue(msg) {
			CountProcessedMessage()
		} else {
			CountNoRouteForMessage()
		}
	}
}
