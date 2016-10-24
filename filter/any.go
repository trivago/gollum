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
	"github.com/trivago/gollum/shared"
)

// Any filter plugin
// This plugin blocks messages after a certain number of messages per second
// has been reached.
// Configuration example
//
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
	filters []core.Filter
}

func init() {
	shared.TypeRegistry.Register(Any{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *Any) Configure(conf core.PluginConfig) error {
	filters := conf.GetStringArray("AnyFilter", []string{})
	for _, filterName := range filters {
		plugin, err := core.NewPluginWithType(filterName, conf)
		if err != nil {
			return err
		}
		f, isFilter := plugin.(core.Filter)
		if !isFilter {
			return fmt.Errorf("%s is not a filter", filterName)
		}
		filter.filters = append(filter.filters, f)
	}

	return nil
}

func (filter *Any) Accepts(msg core.Message) bool {
	for _, f := range filter.filters {
		if f.Accepts(msg) {
			return true
		}
	}
	return false
}
