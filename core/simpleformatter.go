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
	"reflect"

	"github.com/sirupsen/logrus"
)

// SimpleFormatter formatter
//
// This type defines a common baseclass for formatters. Formatter plugins
// may derive from this class.
//
// Parameters
//
// - Source: This value chooses the part of the message the data to be formatted
// should be read from. Use "" to target the message payload; other values
// specify the name of a metadata field to target.
// By default this parameter is set to "".
//
// - Target: This value chooses the part of the message the formatted data
// should be stored to. Use "" to target the message payload; other values
// specify the name of a metadata field to target.
// By default this parameter is set to "".
//
// - SkipIfEmpty: When set to true, this formatter will not be applied to data
// that is empty or - in case of metadata - not existing.
// By default this parameter is set to false
type SimpleFormatter struct {
	Logger                logrus.FieldLogger
	GetSourceData         GetDataFunc
	GetSourceDataAsBytes  GetDataAsBytesFunc
	GetSourceDataAsString GetDataAsStringFunc
	GetTargetData         GetDataFunc
	GetTargetDataAsBytes  GetDataAsBytesFunc
	GetTargetDataAsString GetDataAsStringFunc
	GetTargetAsMetadata   GetMetadataRootFunc
	ForceTargetAsMetadata ForceMetadataRootFunc
	SetTargetData         SetDataFunc
	SetSourceData         SetDataFunc
	SkipIfEmpty           bool `config:"SkipIfEmpty"`
}

// Configure sets up all values required by SimpleFormatter.
func (format *SimpleFormatter) Configure(conf PluginConfigReader) {
	format.Logger = conf.GetSubLogger("Formatter")

	target := conf.GetString("Target", "")
	source := conf.GetString("Source", "")

	format.GetSourceData = NewGetterFor(source)
	format.GetSourceDataAsBytes = NewBytesGetterFor(source)
	format.GetSourceDataAsString = NewStringGetterFor(source)
	format.SetSourceData = NewSetterFor(source)

	format.GetTargetData = NewGetterFor(target)
	format.GetTargetDataAsBytes = NewBytesGetterFor(target)
	format.GetTargetDataAsString = NewStringGetterFor(target)
	format.SetTargetData = NewSetterFor(target)
	format.GetTargetAsMetadata = NewMetadataRootGetterFor(target)
	format.ForceTargetAsMetadata = NewForceMetadataRootGetterFor(target)
}

// CanBeApplied returns true if the formatter can be applied to this message
func (format *SimpleFormatter) CanBeApplied(msg *Message) bool {
	if !format.SkipIfEmpty {
		return true
	}

	data := format.GetSourceData(msg)
	if data == nil {
		return false
	}

	// payload fast path
	if bytes, isBytes := data.([]byte); isBytes {
		return len(bytes) > 0
	}

	// metadata string fast path
	if str, isString := data.(string); isString {
		return len(str) > 0
	}

	// arbitrary, len-able metadata path
	value := reflect.ValueOf(data)
	switch value.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return value.Len() > 0
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
