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

package core

import (
	"github.com/trivago/tgo/tlog"
)

// SimpleFormatter defines the standard formatter implementation.
type SimpleFormatter struct {
	Log tlog.LogScope
}

// SetLogScope sets the log scope to be used for this formatter
func (format *SimpleFormatter) SetLogScope(log tlog.LogScope) {
	format.Log = log
}

// Configure sets up all values requred by SimpleFormatter.
func (format *SimpleFormatter) Configure(conf PluginConfigReader) error {
	format.Log = conf.GetSubLogScope("Formatter")
	return nil
}
