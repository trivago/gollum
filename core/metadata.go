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

// Metadata is a map for optional meta data which can set by consumers and modulators
type Metadata map[string][]byte

// SetValue set a key value pair at meta data
func (meta Metadata) SetValue(key string, value []byte) {
	meta[key] = value
}

// TrySetValue sets a key value pair only if the key is already existing
func (meta Metadata) TrySetValue(key string, value []byte) bool {
	if _, exists := meta[key]; exists {
		meta[key] = value
		return true
	}
	return false
}

// GetValue returns a meta data value by key. This function returns a value if
// key is not set, too. In that case it will return an empty byte array.
func (meta Metadata) GetValue(key string) []byte {
	if value, isSet := meta[key]; isSet {
		return value
	}

	return []byte{}
}

// TryGetValue behaves like GetValue but returns a second value which denotes
// if the key was set or not.
func (meta Metadata) TryGetValue(key string) ([]byte, bool) {
	if value, isSet := meta[key]; isSet {
		return value, true
	}
	return []byte{}, false
}

// GetValueString casts the results of GetValue to a string
func (meta Metadata) GetValueString(key string) string {
	return string(meta.GetValue(key))
}

// TryGetValueString casts the data result of TryGetValue to string
func (meta Metadata) TryGetValueString(key string) (string, bool) {
	data, exists := meta.TryGetValue(key)
	return string(data), exists
}

// Delete removes the given key from the map
func (meta Metadata) Delete(key string) {
	delete(meta, key)
}

// Clone creates an exact copy of this metadata map.
func (meta Metadata) Clone() (clone Metadata) {
	clone = Metadata{}
	for k, v := range meta {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		clone[k] = vCopy
	}
	return
}
