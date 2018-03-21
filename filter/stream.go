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

package filter

import (
	"github.com/trivago/gollum/core"
)

// Stream filter plugin
//
// The "Stream" filter filters messages by applying black and white lists
// to the the messages' streams' names.
//
// The blacklist is applied first; messages not rejected by the blacklist
// are checked against the whitelist. An empty white list matches all
// streams.
//
// Parameters
//
// - Block: Defines a list of stream names that are blocked. If a message's
// stream is not in that list, the "Only" list is tested. By default this
// parameter is empty.
//
// - Only: Defines a list of streams that may pass. Messages from streams
// that are not in this list are blocked unless the list is empty.
// By default this parameter is empty.
//
//
// Examples
//
// This example accepts ALL messages except ones from stream "foo":
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - filter.Stream:
//          Block:
//            - foo
//
// This example only accepts messages from stream "foo":
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - filter.Stream:
//          Only:
//            - foo
//
type Stream struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	blacklist         []core.MessageStreamID
	whitelist         []core.MessageStreamID
}

func init() {
	core.TypeRegistry.Register(Stream{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Stream) Configure(conf core.PluginConfigReader) {
	filter.blacklist = conf.GetStreamArray("Block", []core.MessageStreamID{})
	filter.whitelist = conf.GetStreamArray("Only", []core.MessageStreamID{})
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *Stream) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	for _, blockedID := range filter.blacklist {
		if msg.GetStreamID() == blockedID {
			return filter.GetFilterResultMessageReject(), nil // ### return, explicitly blocked ###
		}
	}

	for _, allowedID := range filter.whitelist {
		if msg.GetStreamID() == allowedID {
			return core.FilterResultMessageAccept, nil // ### return, explicitly allowed ###
		}
	}

	// Return true if no whitlist is given, false otherwise (must fulfill whitelist)
	if len(filter.whitelist) > 0 {
		return filter.GetFilterResultMessageReject(), nil
	}

	return core.FilterResultMessageAccept, nil
}
