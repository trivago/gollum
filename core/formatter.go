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

// A Formatter defines a modification step inside the message
// A Formatter also have to implement the modulator interface
type Formatter interface {
	ApplyFormatter(msg *Message) error
	CanBeApplied(msg *Message) bool
}

// FormatterArray is a type wrapper to []Formatter to make array of formatter
type FormatterArray []Formatter

// CanBeApplied returns true if the array is not empty
func (formatters FormatterArray) CanBeApplied(msg *Message) bool {
	return len(formatters) > 0
}

// ApplyFormatter calls ApplyFormatter on every formatter
func (formatters FormatterArray) ApplyFormatter(msg *Message) error {
	for _, formatter := range formatters {
		if formatter.CanBeApplied(msg) {
			if err := formatter.ApplyFormatter(msg); err != nil {
				return err
			}
		}
	}
	return nil
}
