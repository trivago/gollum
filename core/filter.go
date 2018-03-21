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

// A Filter defines an analysis step inside the message
// A filter also have to implement the modulator interface
type Filter interface {
	ApplyFilter(msg *Message) (FilterResult, error)
}

// FilterArray is a type wrapper to []Filter to make array of filters
type FilterArray []Filter

// ApplyFilter calls ApplyFilter on every filter
// Return FilterResultMessageReject in case of an error or if one filter rejects
func (filters FilterArray) ApplyFilter(msg *Message) (FilterResult, error) {
	for _, filter := range filters {
		result, err := filter.ApplyFilter(msg)
		if err != nil {
			return result, err
		}

		if result != FilterResultMessageAccept {
			return result, nil
		}
	}
	return FilterResultMessageAccept, nil
}
