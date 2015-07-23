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

package filter

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Stream filters messages by stream based on a black and a whitelist.
// The blacklist is checked first.
// Configuration example
//
//   - "stream.Broadcast":
//     Filter: "filter.Stream"
//     FilterBlockStreams:
//       - "foo"
//     FilterOnlyStreams:
//       - "test1"
//       - "test2"
//
// FilterBlockStreams sets a list of streams that are blocked. If a message's
// stream is not in that list, the OnlyStreams list is tested. This list ist
// empty by default.
//
// FilterOnlyStreams sets a list of streams that may pass. Messages from streams
// that are not in this list are blocked unless the list is empty. By default
// this list is empty.
type Stream struct {
	blacklist []core.MessageStreamID
	whitelist []core.MessageStreamID
}

func init() {
	shared.RuntimeType.Register(Stream{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Stream) Configure(conf core.PluginConfig) error {
	filter.blacklist = conf.GetStreamArray("FilterBlockStreams", []core.MessageStreamID{})
	filter.whitelist = conf.GetStreamArray("FilterOnlyStreams", []core.MessageStreamID{})
	return nil
}

// Accepts filters by streamId using a black and whitelist
func (filter *Stream) Accepts(msg core.Message) bool {
	for _, blockID := range filter.blacklist {
		if msg.StreamID == blockID {
			return false // ### return, explicitly blocked ###
		}
	}

	for _, passID := range filter.whitelist {
		if msg.StreamID == passID {
			return true // ### return, explicitly allowed ###
		}
	}

	// Return true if no whitlist is given, false otherwise (must fullfill whitelist)
	return len(filter.whitelist) == 0
}
