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

package filter

import (
	"github.com/trivago/gollum/core"
)

// Stream filter plugin
// This plugin filters messages by stream based on a black and a whitelist.
// The blacklist is checked first.
// Configuration example
//
//  - "stream.Broadcast":
//    Filter: "filter.Stream"
//    FilterBlockStreams:
//      - "foo"
//    FilterOnlyStreams:
//      - "test1"
//      - "test2"
//
// FilterBlockStreams sets a list of streams that are blocked. If a message's
// stream is not in that list, the OnlyStreams list is tested. This list ist
// empty by default.
//
// FilterOnlyStreams sets a list of streams that may pass. Messages from streams
// that are not in this list are blocked unless the list is empty. By default
// this list is empty.
type Stream struct {
	core.SimpleFilter
	blacklist []core.MessageStreamID
	whitelist []core.MessageStreamID
}

func init() {
	core.TypeRegistry.Register(Stream{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Stream) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)

	filter.blacklist = conf.GetStreamArray("Block", []core.MessageStreamID{})
	filter.whitelist = conf.GetStreamArray("Only", []core.MessageStreamID{})

	return conf.Errors.OrNil()
}

// Modulate filters by streamId using a black and whitelist
func (filter *Stream) Modulate(msg *core.Message) core.ModulateResult {
	for _, blockedID := range filter.blacklist {
		if msg.StreamID() == blockedID {
			return filter.Drop(msg) // ### return, explicitly blocked ###
		}
	}

	for _, allowedID := range filter.whitelist {
		if msg.StreamID() == allowedID {
			return core.ModulateResultContinue // ### return, explicitly allowed ###
		}
	}

	// Return true if no whitlist is given, false otherwise (must fulfill whitelist)
	if len(filter.whitelist) > 0 {
		return filter.Drop(msg)
	}

	return core.ModulateResultContinue
}
