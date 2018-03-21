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

// SimpleFilter plugin base type
//
// This type defines a common base class for all Filters. All filter plugins
// should derive from this class but don't necessarily need to.
//
// Parameters
//
// - FilteredStream: This value defines the stream filtered messages get sent to.
// You can disable this behavior by setting the value to "".
// By default this parameter is set to "".
//
type SimpleFilter struct {
	Logger           logrus.FieldLogger
	filteredStreamID MessageStreamID `config:"FilteredStream"`
}

// SetLogger sets the scoped logger to be used for this filter
func (filter *SimpleFilter) SetLogger(logger logrus.FieldLogger) {
	filter.Logger = logger
}

// Configure sets up all values required by SimpleFormatter.
func (filter *SimpleFilter) Configure(conf PluginConfigReader) error {
	filter.Logger = conf.GetSubLogger("Filter")
	//filter.filteredStreamID = GetStreamID(conf.GetString("FilteredStream", InvalidStream))
	return nil
}

// GetLogger returns this plugin's scoped logger
func (filter *SimpleFilter) GetLogger() logrus.FieldLogger {
	return filter.Logger
}

// GetFilterResultMessageReject returns a FilterResultMessageReject with the
// stream set to GetfilteredStreamID()
func (filter *SimpleFilter) GetFilterResultMessageReject() FilterResult {
	return FilterResultMessageReject(filter.filteredStreamID)
}
