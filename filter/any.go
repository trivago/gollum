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
// AnyFilter defines a list of filters that should be checked before dropping
// a message. Filters are checked in order, and if the message passes
// then no further filters are checked. By default this list is empty.
type Any struct {
	core.SimpleFilter
	modulators core.ModulatorArray
}

func init() {
	core.TypeRegistry.Register(Any{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Any) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)

	filter.modulators = conf.GetModulatorArray("Any",
		filter.Log, core.ModulatorArray{})

	return conf.Errors.OrNil()
}

// Accepts allows messages where at least one nested filter returns true
// todo: review function!!!
func (filter *Any) Modulate(msg *core.Message) core.ModulateResult {
	for _, modulator := range filter.modulators {
		if res := modulator.Modulate(msg); res != core.ModulateResultDiscard {
			return res
		}
	}

	return filter.Drop(msg)
}

func (filter *Any) HasToFilter(msg *core.Message) (bool, error) {
	//for _, filter := range filter.modulators {
	//	if res := filter.Has(msg); res != core.ModulateResultDiscard {
	//		return res
	//	}
	//}
	//
	//return filter.Drop(msg)
	return false, nil
}