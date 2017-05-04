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

// SimpleFilter plugin base type
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
type SimpleFilter struct {
	Log          tlog.LogScope
	dropStreamID MessageStreamID
}

// SetLogScope sets the log scope to be used for this filter
func (filter *SimpleFilter) SetLogScope(log tlog.LogScope) {
	filter.Log = log
}

// Configure sets up all values requred by SimpleFormatter.
func (filter *SimpleFilter) Configure(conf PluginConfigReader) error {
	filter.Log = conf.GetSubLogScope("Filter")

	filter.dropStreamID = GetStreamID(conf.GetString("DropToStream", InvalidStream))
	return nil
}

// GetDropStreamID return the drop stream id
func (filter *SimpleFilter) GetDropStreamID() MessageStreamID {
	return filter.dropStreamID
}

// MessageRejectResult returns a FilterResultMessageReject with the stream set to
// GetDropStreamID()
func (filter *SimpleFilter) MessageRejectResult() FilterResult {
	return FilterResultMessageReject(filter.dropStreamID)
}
