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
	"github.com/trivago/gollum/core"
)

// Any filter plugin
//
// This plugin takes a list of filters and applies each of them to incoming
// messages until a an accepting filter is found. If any of the listed filters
// accept the message, it is passed through, otherwise, the message is dropper.
//
// Parameters
//
// - AnyFilters: Defines a list of filters that should be checked before filtering
// a message. Filters are checked in order, and if the message passes
// then no further filters are checked.
//
// Examples
//
// This example will accept valid json or messages from "exceptionStream":
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - filter.Any:
//        AnyFilters:
//          - filter.JSON
//          - filter.Stream:
//            Only: exceptionStream
type Any struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	filters           core.FilterArray `config:"AnyFilters"`
}

func init() {
	core.TypeRegistry.Register(Any{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Any) Configure(conf core.PluginConfigReader) {
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *Any) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	for _, subFilter := range filter.filters {
		// all filter need to apply the message. if one filter not apply return FilterResultMessageAccept
		if res, _ := subFilter.ApplyFilter(msg); res == core.FilterResultMessageAccept {
			return core.FilterResultMessageAccept, nil
		}
	}

	return filter.GetFilterResultMessageReject(), nil
}
