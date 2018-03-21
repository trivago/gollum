// Copyright 2015-2018 trivago N.V.
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
	"github.com/sirupsen/logrus"
)

// SimpleFormatter formatter
//
// This type defines a common baseclass for formatters. Formatter plugins
// may derive from this class.
//
// Parameters
//
// - ApplyTo: This value chooses the part of the message the formatting
// should be applied to. Use "" to target the message payload; other values
// specify the name of a metadata field to target.
// By default this parameter is set to "".
//
// - SkipIfEmpty: When set to true, this formatter will not be applied to data
// that is empty or - in case of metadata - not existing.
// By default this parameter is set to false
type SimpleFormatter struct {
	Logger            logrus.FieldLogger
	GetAppliedContent GetAppliedContent
	SetAppliedContent SetAppliedContent
	SkipIfEmpty       bool `config:"SkipIfEmpty"`
}

// Configure sets up all values required by SimpleFormatter.
func (format *SimpleFormatter) Configure(conf PluginConfigReader) {
	format.Logger = conf.GetSubLogger("Formatter")

	applyTo := conf.GetString("ApplyTo", "")
	format.GetAppliedContent = GetAppliedContentGetFunction(applyTo)
	format.SetAppliedContent = GetAppliedContentSetFunction(applyTo)
}

// CanBeApplied returns true if the formatter can be applied to this message
func (format *SimpleFormatter) CanBeApplied(msg *Message) bool {
	if format.SkipIfEmpty {
		return len(format.GetAppliedContent(msg)) > 0
	}
	return true
}

// SetLogger sets the scoped logger to be used for this formatter
func (format *SimpleFormatter) SetLogger(logger logrus.FieldLogger) {
	format.Logger = logger
}

// GetLogger returns the scoped logger of this plugin
func (format *SimpleFormatter) GetLogger() logrus.FieldLogger {
	return format.Logger
}
