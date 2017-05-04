// Copyright 2015-2017 trivago GmbH
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

// None filter plugin
// This plugin blocks all messages.
// Configuration example
//
//   - "stream.Broadcast":
//     Filter: "filter.None"
//
type None struct {
	core.SimpleFilter
}

func init() {
	core.TypeRegistry.Register(None{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *None) Configure(conf core.PluginConfigReader) error {
	return filter.SimpleFilter.Configure(conf)
}

// ApplyFilter calls the Filter.ApplyFilter method
func (filter *None) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	return core.FilterResultMessageReject, nil
}