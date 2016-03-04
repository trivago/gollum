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
	// Accepts returns true if this filter validated the given message (pass)
	Accepts(msg Message) bool

	// SetLogScope sets the log scope to be used for this filter
	SetLogScope(log tlog.LogScope)

	// Drop sends the given message to the stream configured with this filter.
	// This method is called by the framework after Accepts failed.
	Drop(msg Message)
}

// FilterFunc is the function signature type used by all filter functions.
type FilterFunc func(msg Message) bool

// FilterBase plugin base type
// This type defines a common baseclass for all Filters. All filter plugins
// should derive from this class but don't necessarily need to.
// Configuration example:
//
//  - "plugin":
//    Filters:
//      - SomeFilter:
//        DropToStream: ""
//
// DropToStream defines a stream where filtered messages get sent to.
// You can disable this behavior by setting "". Set to "" by default.
type FilterBase struct {
	Log          tlog.LogScope
	dropStreamID MessageStreamID
}

// SetLogScope sets the log scope to be used for this filter
func (filter *FilterBase) SetLogScope(log tlog.LogScope) {
	filter.Log = log
}

// Configure sets up all values requred by FormatterBase.
func (filter *FilterBase) Configure(conf PluginConfigReader) error {
	filter.Log = conf.GetSubLogScope("Filter")
	filter.dropStreamID = GetStreamID(conf.GetString("DropToStream", InvalidStream))
	return nil
}

// Drop sends the given message to the stream configured with this filter.
func (filter *FilterBase) Drop(msg Message) {
	if filter.dropStreamID != InvalidStreamID {
		msg.Route(filter.dropStreamID)
	}
}
