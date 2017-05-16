// Copyright 2015-2017 trivago GmbH
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

// Router defines the interface for all stream plugins
type Router interface {
	Modulator

	// StreamID returns the stream id this plugin is bound to.
	StreamID() MessageStreamID

	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Enqueue sends a given message to all registered end points.
	// This function is called by Route() which should be preferred over this
	// function when sending messages.
	Enqueue(msg *Message) error

	// Start starts the router by the coordinator.StartPlugins() method
	Start() error
}

// Route tries to enqueue a message to the given stream. This function also
// handles redirections enforced by formatters.
func Route(msg *Message, router Router) error {
	action := router.Modulate(msg)

	switch action {
	case ModulateResultDiscard:
		DiscardMessage(msg)
		return nil

	case ModulateResultContinue:
		return router.Enqueue(msg)

	case ModulateResultFallback:
		if msg.StreamID() == router.StreamID() {
			streamName := StreamRegistry.GetStreamName(msg.StreamID())
			prevStreamName := StreamRegistry.GetStreamName(msg.PreviousStreamID())
			return NewModulateResultError("Routing loop detected for router %s (from %s)", streamName, prevStreamName)
		}

		return RouteOriginal(msg, msg.GetRouter())
	}

	return NewModulateResultError("Unknown ModulateResult action: %d", action)
}

// RouteOriginal restores the original message and routes it by using a
// a given router.
func RouteOriginal(msg *Message, router Router) error {
	return Route(msg.CloneOriginal(), router)
}

// DiscardMessage increases the discard statistic and discards the given
// message.
func DiscardMessage(msg *Message) {
	CountDiscardedMessage()
}
