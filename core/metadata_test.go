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
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestMetadataSetGet(t *testing.T) {
	expect := ttesting.NewExpect(t)

	meta := make(Metadata)

	meta.SetValue("foo", []byte("foo_value"))
	expect.Equal([]byte("foo_value"), meta.GetValue("foo"))

	setExisiting := meta.TrySetValue("foo", []byte("foo_value"))
	expect.True(setExisiting)

	setExisiting = meta.TrySetValue("bar", []byte("bar_value"))
	expect.False(setExisiting)

	barValue, exists := meta.TryGetValue("bar")
	expect.False(exists)
	expect.Equal([]byte{}, barValue)

	barStrValue, exists := meta.TryGetValueString("bar")
	expect.False(exists)
	expect.Equal("", barStrValue)

	fooVal, exists := meta.TryGetValue("foo")
	expect.True(exists)
	expect.Equal([]byte("foo_value"), fooVal)

	fooStrVal, exists := meta.TryGetValueString("foo")
	expect.True(exists)
	expect.Equal("foo_value", fooStrVal)

	meta2 := meta.Clone()

	meta.Delete("foo")
	_, exists = meta.TryGetValue("foo")
	expect.False(exists)

	_, exists = meta2.TryGetValue("foo")
	expect.True(exists)
}
