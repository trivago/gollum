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

package core

import (
	"github.com/trivago/tgo/tlog"
)

// Filter allows custom message filtering for ProducerBase derived plugins.
// Producers not deriving from ProducerBase might utilize this one, too.
type Filter interface {
	Accepts(msg Message) bool
}

// FilterFunc is the function signature type used by all filter functions.
type FilterFunc func(msg Message) bool

// FilterBase defines the standard filter implementation.
type FilterBase struct {
	Log tlog.LogScope
}

// Configure sets up all values requred by FormatterBase.
func (filter *FilterBase) Configure(conf PluginConfig) error {
	filter.Log = NewSubPluginLogScope(conf, "Filter")
	return nil
}
