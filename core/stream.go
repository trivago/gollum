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

// Stream defines the interface for all stream plugins
type Stream interface {
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
}

// Route tries to enqueue a message to the given stream. This function also
// handles redirections enforced by formatters.
func Route(msg *Message, stream Stream) error {
	action := stream.Modulate(msg)

	switch action {
	case ModulateResultDiscard:
		CountDiscardedMessage()
		return nil

	case ModulateResultContinue:
		return stream.Enqueue(msg)

	case ModulateResultRoute, ModulateResultDrop:
		if msg.StreamID() == stream.StreamID() {
			streamName := StreamRegistry.GetStreamName(msg.StreamID())
			prevStreamName := StreamRegistry.GetStreamName(msg.PreviousStreamID())
			return NewModulateResultError("Routing loop detected for stream %s (from %s)", streamName, prevStreamName)
		}

		return Route(msg, msg.GetStream())
	}

	return NewModulateResultError("Unknown ModulateResult action: %d", action)
}
