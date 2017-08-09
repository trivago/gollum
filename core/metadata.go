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

// Metadata is a map for optional meta data which can set by consumers and modulators
type Metadata map[string][]byte

// SetValue set a key value pair at meta data
func (meta Metadata) SetValue(key string, value []byte) {
	meta[key] = value
}

// GetValue returns a meta data value by key
func (meta Metadata) GetValue(key string) []byte {
	if value, isSet := meta[key]; isSet {
		return value
	}

	return []byte{}
}

// GetValueString returns the meta value by GetValue as string
func (meta Metadata) GetValueString(key string) string {
	return string(meta.GetValue(key))
}

// Delete delete a meta data value by key
func (meta Metadata) Delete(key string) {
	delete(meta, key)
}

// HasValue returns true if the given key exists
func (meta Metadata) HasValue(key string) bool {
	_, exists := meta[key]
	return exists
}

// Clone Metadata byte values to new Metadata map
func (meta Metadata) Clone() (clone Metadata) {
	clone = Metadata{}
	for k, v := range meta {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		clone[k] = vCopy
	}

	return
}
