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

// FilterArray is a type wrapper to []Filter to make array of filters
type FilterArray []Filter

// A Filter defines an analysis step inside the message
// A filter also have to implement the modulator interface
type Filter interface {
	Modulator
	HasToFilter (msg *Message) (bool, error)
}