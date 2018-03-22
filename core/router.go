// Copyright 2015-2018 trivago N.V.
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
	"fmt"
	"time"
)

// Router defines the interface for all stream plugins
type Router interface {
	Modulator

	// StreamID returns the stream id this plugin is bound to.
	GetStreamID() MessageStreamID

	// GetID returns the pluginID of the message source
	GetID() string

	// AddProducer adds one or more producers to this stream, i.e. the producers
	// listening to messages on this stream.
	AddProducer(producers ...Producer)

	// Enqueue sends a given message to all registered end points.
	// This function is called by Route() which should be preferred over this
	// function when sending messages.
	Enqueue(msg *Message) error

	// GetTimeout returns the timeout configured for this router
	GetTimeout() time.Duration

	// Start starts the router by the coordinator.StartPlugins() method
	Start() error
}

// Route tries to enqueue a message to the given stream. This function also
// handles redirections enforced by formatters.
func Route(msg *Message, router Router) error {
	if router == nil {
		DiscardMessage(msg, "nil", fmt.Sprintf("Router for stream %s is nil", msg.GetStreamID().GetName()))
		return nil
	}

	action := router.Modulate(msg)

	streamName := msg.GetStreamID().GetName()
	streamMetric := GetStreamMetric(msg.GetStreamID())

	switch action {
	case ModulateResultDiscard:
		streamMetric.CountMessageDiscarded()
		DiscardMessage(msg, router.GetID(), "Router discarded")
		return nil

	case ModulateResultContinue:
		streamMetric.CountMessageRouted()
		CountMessageRouted()
		MessageTrace(msg, router.GetID(), "Routed")

		return router.Enqueue(msg)

	case ModulateResultFallback:
		if msg.GetStreamID() == router.GetStreamID() {

			prevStreamName := StreamRegistry.GetStreamName(msg.GetPrevStreamID())
			return NewModulateResultError("Routing loop detected for router %s (from %s)", streamName, prevStreamName)
		}

		// Do NOT route the original message in this case.
		// If a modulate pipeline returns fallback, the message might have been
		// modified already. To meet expectations the CHANGED message has to be
		// routed.

		return Route(msg, msg.GetRouter())
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
func DiscardMessage(msg *Message, pluginID string, comment string) {
	CountMessageDiscarded()
	MessageTrace(msg, pluginID, comment)
}
