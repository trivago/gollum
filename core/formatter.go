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

// Formatter is the interface definition for message formatters
type Formatter interface {
	// Format transfers the message payload into a new format. The payload may
	// then be reassigned to the original or a new message.
	// In addition to that the formatter may change the stream of the message.
	Format(msg *Message) ([]byte, MessageStreamID)

	// SetLogScope sets the log scope to be used for this formatter
	SetLogScope(log tlog.LogScope)
}

// FormatterFunc is the function signature type used by all formating functions.
type FormatterFunc func(msg *Message) ([]byte, MessageStreamID)

// FormatterBase defines the standard formatter implementation.
type FormatterBase struct {
	Log tlog.LogScope
}

// SetLogScope sets the log scope to be used for this formatter
func (format *FormatterBase) SetLogScope(log tlog.LogScope) {
	format.Log = log
}

// Configure sets up all values requred by FormatterBase.
func (format *FormatterBase) Configure(conf PluginConfigReader) error {
	format.Log = conf.GetSubLogScope("Formatter")
	return nil
}
