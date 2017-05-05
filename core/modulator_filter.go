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

// FilterResult defines a set of results for filtering
type FilterResult int

const (
	// FilterResultMessageReject indicates that a message is filtered and not pared along the stream
	FilterResultMessageReject = FilterResult(iota)
	// FilterResultMessageAccept indicates that a message can be passed along and continue
	FilterResultMessageAccept = FilterResult(iota)
)

// FilterArray is a type wrapper to []Filter to make array of filters
type FilterArray []Filter

// ApplyFilter calls ApplyFilter on every filter
// Return FilterResultMessageReject in case of an error or if one filter rejects
func (filters FilterArray) ApplyFilter(msg *Message) (FilterResult, error) {
	for _, filter := range filters {
		result, err := filter.ApplyFilter(msg)
		if err != nil {
			return FilterResultMessageReject, err
		}

		if result == FilterResultMessageReject {
			return FilterResultMessageReject, nil
		}
	}
	return FilterResultMessageAccept, nil
}

// A Filter defines an analysis step inside the message
// A filter also have to implement the modulator interface
type Filter interface {
	ApplyFilter(msg *Message) (FilterResult, error)
	GetDropStreamID() MessageStreamID
}

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
		tlog.Warning.Print("FilterModulator with error:", err)
		return filterModulator.drop(msg)
	}

	switch result {
	case FilterResultMessageAccept:
		return ModulateResultContinue
	case FilterResultMessageReject:
		return filterModulator.drop(msg)
	default:
		tlog.Error.Printf("FilterModulator '%T' with unknown return value: %d",
			filterModulator.Filter, result)
		return ModulateResultContinue
	}
}

// ApplyFilter calls the Filter.ApplyFilter method
func (filterModulator *FilterModulator) ApplyFilter(msg *Message) (FilterResult, error) {
	return filterModulator.Filter.ApplyFilter(msg)
}

// drop set filter DropStreamID to message and returns ModulateResult
func (filterModulator *FilterModulator) drop(msg *Message) ModulateResult {
	// if not drop stream => discard | else drop
	filterDropStreamID := filterModulator.Filter.GetDropStreamID()
	if filterDropStreamID != InvalidStreamID {
		msg.SetStreamID(filterDropStreamID)
		return ModulateResultDrop
	}
	return ModulateResultDiscard
}
