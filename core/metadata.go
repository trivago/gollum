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

// MetaData is a map for optional meta data which can set by consumers and modulators
type MetaData map[string][]byte

// SetValue set a key value pair at meta data
func (meta MetaData) SetValue(key string, value []byte) {
	meta[key] = value
}

// GetValue returns a meta data value by key
func (meta MetaData) GetValue(key string) []byte {
	if value, isSet := meta[key]; isSet {
		return value
	}

	return []byte{}
}

// GetValueString returns the meta value by GetValue as string
func (meta MetaData) GetValueString(key string) string {
	return string(meta.GetValue(key))
}

// ResetValue delete a meta data value by key
func (meta MetaData) ResetValue(key string) {
	delete(meta, key)
}

// Clone MetaData byte values to new MetaData map
func (meta MetaData) Clone() MetaData {
	clone := MetaData{}
	for k, v := range meta {
		vCopy := make([]byte, len(v))
		copy(vCopy, v)
		clone[k] = vCopy
	}

	return clone
}
