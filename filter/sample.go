// Copyright 2015-2018 trivago N.V.
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
//
// This plugin can be used to get n out of m messages (downsample).
// This allows you to reduce the amount of messages; the plugin starts
// blocking after a certain number of messages has been reached.
//
// Parameters
//
// - SampleRatePerGroup: This value defines how many messages are passed through
// the filter in each group.
// By default this parameter is set to "1".
//
// - SampleGroupSize: This value defines how many messages make up a group. Messages over
// SampleRatePerGroup within a group are filtered.
// By default this parameter is set to "2".
//
// - SampleRateIgnore: This value defines a list of streams that should not be affected by
// sampling. This is useful for e.g. producers listening to "*".
// By default this parameter is set to an empty list.
//
// Examples
//
// This example will block 8 from 10 messages:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - filter.Sample:
//        SampleRatePerGroup: 2
//        SampleGroupSize: 10
//        SampleIgnore:
//          - foo
//          - bar
//
type Sample struct {
	core.SimpleFilter
	rate   uint64 `config:"SampleRatePerGroup" default:"1"`
	group  uint64 `config:"SampleGroupSize" default:"2"`
	count  *uint64
	ignore map[core.MessageStreamID]bool
}

func init() {
	core.TypeRegistry.Register(Sample{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Sample) Configure(conf core.PluginConfigReader) {
	filter.count = new(uint64)
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

	// Accept the first n messages of each group, reject the rest
	// Overflow is not really an issue here as it will take years to get one
	index := (atomic.AddUint64(filter.count, 1) - 1) % filter.group
	if index < filter.rate {
		return core.FilterResultMessageAccept, nil // ### return, ok ###
	}

	return filter.GetFilterResultMessageReject(), nil
}
