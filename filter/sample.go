// Copyright 2015-2017 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	 http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filter

import (
	"sync/atomic"

	"github.com/trivago/gollum/core"
)

// Sample filter plugin
// This plugin blocks messages after a certain number of messages per second
// has been reached.
// Configuration example
//
//   - "stream.Broadcast":
//	   Filter: "filter.Sample"
//	   SampleRatePerGroup: 1
//	   SampleGroupSize: 1
//	   SampleRateIgnore:
//	     - "foo"
//
// SampleRatePerGroup defines how many messages are passed through the filter
// in each group. By default this is set to 1.
//
// SampleGroupSize defines how many messages make up a group. Messages over
// SampleRatePerGroup within a group are filtered. By default this is set to 1.
//
// SampleRateIgnore defines a list of streams that should not be affected by
// sampling. This is useful for e.g. producers listeing to "*".
// By default this list is empty.
type Sample struct {
	core.SimpleFilter
	rate   int64
	group  int64
	count  *int64
	ignore map[core.MessageStreamID]bool
}

func init() {
	core.TypeRegistry.Register(Sample{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Sample) Configure(conf core.PluginConfigReader) {
	filter.rate = int64(conf.GetInt("SampleRatePerGroup", 1))
	filter.group = int64(conf.GetInt("SampleGroupSize", 1))
	filter.count = new(int64)

	filter.ignore = make(map[core.MessageStreamID]bool)
	ignore := conf.GetStreamArray("SampleIgnore", []core.MessageStreamID{})
	for _, stream := range ignore {
		filter.ignore[stream] = true
	}
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *Sample) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	// Ignore based on StreamID
	if ignore, known := filter.ignore[msg.GetStreamID()]; known && ignore {
		return core.FilterResultMessageAccept, nil // ### return, do not limit ###
	}

	// Check if count needs to be reset
	count := atomic.AddInt64(filter.count, 1)
	if count > filter.group {
		if count%filter.group == 1 {
			// make sure we never overflow filter.count
			count = atomic.AddInt64(filter.count, -(filter.group)) // TODO: filter.count can be != count here (!)
		} else {
			// range from 1 to filter.group
			count = (count-1)%filter.group + 1
		}
	}

	// Check if to be filtered
	if count > filter.rate {
		return filter.GetFilterResultMessageReject(), nil // ### return, filter ###
	}

	return core.FilterResultMessageAccept, nil
}
