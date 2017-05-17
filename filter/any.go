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
	"github.com/trivago/gollum/core"
)

// Any filter plugin
// This plugin blocks messages after a certain number of messages per second
// has been reached.
// Configuration example
//	- "yourStreamId":
//	  type: "stream.Broadcast"
//	    filters:
//	      - "filter.Any"
//	      Any:
//		- "filter.JSON"
//		- "filter.RegEx"
//
// AnyFilter defines a list of filters that should be checked before filtering
// a message. Filters are checked in order, and if the message passes
// then no further filters are checked. By default this list is empty.
type Any struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	filters core.FilterArray
}

func init() {
	core.TypeRegistry.Register(Any{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Any) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)

	filter.filters = conf.GetFilterArray("Any",
		filter.Log, core.FilterArray{})

	return conf.Errors.OrNil()
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *Any) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	for _, subfilter := range filter.filters {
		// all filter need to apply the message. if one filter not apply return FilterResultMessageAccept
		if res, _ := subfilter.ApplyFilter(msg); res == core.FilterResultMessageAccept {
			return core.FilterResultMessageAccept, nil
		}
	}

	return filter.GetFilterResultMessageReject(), nil
}
