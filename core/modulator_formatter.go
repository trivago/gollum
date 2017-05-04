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

// FormatterArray is a type wrapper to []Formatter to make array of formatter
type FormatterArray []Formatter

// ApplyFormatter calls ApplyFormatter on every formatter
func (formatters FormatterArray) ApplyFormatter(msg *Message) error {
	for _, formatter := range formatters {
		err := formatter.ApplyFormatter(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// A Formatter defines a modification step inside the message
// A Formatter also have to implement the modulator interface
type Formatter interface {
	ApplyFormatter (msg *Message) (error)
}

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

//  Modulate implementation for Formatter
func (formatterModulator *FormatterModulator) Modulate(msg *Message) ModulateResult {
	err := formatterModulator.ApplyFormatter(msg)
	if err != nil {
		tlog.Warning.Print("FormatterModulator with error:", err)
		return ModulateResultDiscard
	}

	return ModulateResultContinue
}

// ApplyFormatter calls the Formatter.ApplyFormatter method
func (formatterModulator *FormatterModulator) ApplyFormatter(msg *Message) error {
	return formatterModulator.Formatter.ApplyFormatter(msg)
}