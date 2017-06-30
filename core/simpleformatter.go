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
	"github.com/sirupsen/logrus"
)

// SimpleFormatter defines the standard formatter implementation.
//
// Configuration example:
//   N/A - placeholder for docs generator
//
// ApplyTo chooses the part of the message the formatting should be
// applied to. Use "payload"  or "" to target the message payload;
// othe values specify the name of a metadata field to target.
// Default "".
type SimpleFormatter struct {
	Logger            logrus.FieldLogger
	GetAppliedContent GetAppliedContent
	SetAppliedContent SetAppliedContent
}

// Configure sets up all values required by SimpleFormatter.
func (format *SimpleFormatter) Configure(conf PluginConfigReader) {
	format.Logger = conf.GetSubLogger("Formatter")

	applyTo := conf.GetString("ApplyTo", "")
	format.GetAppliedContent = GetAppliedContentGetFunction(applyTo)
	format.SetAppliedContent = GetAppliedContentSetFunction(applyTo)
}

// SetLogger sets the scoped logger to be used for this formatter
func (format *SimpleFormatter) SetLogger(logger logrus.FieldLogger) {
	format.Logger = logger
}

// GetLogger returns the scoped logger of this plugin
func (format *SimpleFormatter) GetLogger() logrus.FieldLogger {
	return format.Logger
}
