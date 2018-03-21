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

// FilterResult defines a set of results for filtering
type FilterResult uint64

const (
	// FilterResultMessageAccept indicates that a message can be passed along and continue
	FilterResultMessageAccept = FilterResult(0)
)

// FilterResultMessageReject indicates that a message is filtered and not pared along the stream
func FilterResultMessageReject(stream MessageStreamID) FilterResult {
	return FilterResult(stream)
}

// GetStreamID converts a FilterResult to a stream id
func (f FilterResult) GetStreamID() MessageStreamID {
	if f == FilterResultMessageAccept {
		return InvalidStreamID
	}
	return MessageStreamID(f)
}
