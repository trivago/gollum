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
	"testing"

	"github.com/trivago/tgo/ttesting"
)

func TestMetadataSetGet(t *testing.T) {
	expect := ttesting.NewExpect(t)

	meta := NewMetadata()

	meta.Set("foo", "foo_value")

	val, err := meta.String("foo")
	expect.NoError(err)
	expect.Equal("foo_value", val)

	meta2 := meta.Clone()

	meta.Set("foo", "bar_value")

	val, err = meta.String("foo")
	expect.NoError(err)
	expect.Equal("bar_value", val)

	ifaceVal, exists := meta.Value("bar")
	expect.False(exists)
	expect.Equal(nil, ifaceVal)

	meta.Delete("foo")

	ifaceVal, exists = meta.Value("foo")
	expect.False(exists)
	expect.Equal(nil, ifaceVal)

	ifaceVal, exists = meta2.Value("foo")
	expect.True(exists)
	expect.Equal("foo_value", ifaceVal.(string))
}
