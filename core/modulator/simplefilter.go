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

package modulator

import (
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/gollum/core"
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
	dropStreamID core.MessageStreamID
}

// SetLogScope sets the log scope to be used for this filter
func (filter *SimpleFilter) SetLogScope(log tlog.LogScope) {
	filter.Log = log
}

// Configure sets up all values requred by SimpleFormatter.
func (filter *SimpleFilter) Configure(conf core.PluginConfigReader) error {
	filter.Log = conf.GetSubLogScope("Filter")

	filter.dropStreamID = core.GetStreamID(conf.GetString("DropToStream", core.InvalidStream))
	return nil
}

// Drop sends the given message to the stream configured with this filter.
func (filter *SimpleFilter) Drop(msg *core.Message) core.ModulateResult {
	if filter.dropStreamID != core.InvalidStreamID {
		msg.SetStreamID(filter.dropStreamID)
		return core.ModulateResultDrop
	}
	return core.ModulateResultDiscard
}
