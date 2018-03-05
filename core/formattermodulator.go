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

// FormatterModulator is a wrapper to provide a Formatter as a Modulator
type FormatterModulator struct {
	Formatter Formatter
}

// NewFormatterModulator return a instance of FormatterModulator
func NewFormatterModulator(formatter Formatter) *FormatterModulator {
	return &FormatterModulator{
		Formatter: formatter,
	}
}

// Modulate implementation for Formatter
func (formatterModulator *FormatterModulator) Modulate(msg *Message) ModulateResult {
	err := formatterModulator.ApplyFormatter(msg)
	if err != nil {
		logrus.Warning("FormatterModulator with error:", err)
		return ModulateResultDiscard
	}

	return ModulateResultContinue
}

// CanBeApplied returns true if the array is not empty
func (formatterModulator *FormatterModulator) CanBeApplied(msg *Message) bool {
	return formatterModulator.Formatter.CanBeApplied(msg)
}

// ApplyFormatter calls the Formatter.ApplyFormatter method
func (formatterModulator *FormatterModulator) ApplyFormatter(msg *Message) error {
	if formatterModulator.CanBeApplied(msg) {
		return formatterModulator.Formatter.ApplyFormatter(msg)
	}
	return nil
}
