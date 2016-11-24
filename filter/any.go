// Copyright 2015-2016 trivago GmbH
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
	"fmt"

	"github.com/trivago/gollum/core"
)

// Any filter plugin
// This plugin blocks messages after a certain number of messages per second
// has been reached.
// Configuration example
//	- "yourStreamId"
//		type: "stream.Broadcast"
//		filters:
//			- "filter.Any"
//				Any:
//					- "filter.JSON"
//					- "filter.RegEx"
//   - "stream.Broadcast":
//	 Filter: "filter.Any"
//	 AnyFilter:
//	 - "filter.JSON"
//	 - "filter.RegEx"
//
// AnyFilter defines a list of filters that should be checked before dropping
// a message. Filters are checked in order, and if the message passes
// then no further filters are checked. By default this list is empty.
type Any struct {
	core.SimpleFilter
	filters []core.SimpleFilter
}

func init() {
	core.TypeRegistry.Register(Any{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Any) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)

	filters := conf.GetStringArray("Any", []string{})
	for _, filterName := range filters {
		cfg := core.NewPluginConfig("_Any_"+filterName, filterName)

		plugin, err := core.NewPlugin(core.NewPluginConfigReader(&cfg))
		if err != nil {
			return err
		}

		f, isFilter := plugin.(core.SimpleFilter)
		if !isFilter {
			return fmt.Errorf("%s is not a filter", filterName)
		}

		filter.filters = append(filter.filters, f)
	}

	return conf.Errors.OrNil()
}

func (filter *Any) Modulate(msg *Message) core.ModulateResult {
	for _, f := range filter.filters {
		if res := f.Modulate(msg); res == core.ModulateResultContinue {
			return core.ModulateResultContinue
		}
	}

	return filter.Drop(msg)
}
