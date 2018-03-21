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

// FilterModulator is a wrapper to provide a Filter as a Modulator
type FilterModulator struct {
	Filter Filter
}

// NewFilterModulator return a instance of FilterModulator
func NewFilterModulator(filter Filter) *FilterModulator {
	return &FilterModulator{
		Filter: filter,
	}
}

// Modulate implementation for Filters
func (filterModulator *FilterModulator) Modulate(msg *Message) ModulateResult {
	result, err := filterModulator.ApplyFilter(msg)
	if err != nil {
		logrus.Warning("FilterModulator with error:", err)
	}

	if result == FilterResultMessageAccept {
		return ModulateResultContinue
	}

	newStreamID := result.GetStreamID()
	if newStreamID == InvalidStreamID {
		return ModulateResultDiscard
	}

	msg.SetStreamID(newStreamID)
	return ModulateResultFallback
}

// ApplyFilter calls the Filter.ApplyFilter method
func (filterModulator *FilterModulator) ApplyFilter(msg *Message) (FilterResult, error) {
	return filterModulator.Filter.ApplyFilter(msg)
}
