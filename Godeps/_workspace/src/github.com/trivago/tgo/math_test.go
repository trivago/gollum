// Copyright 2015 trivago GmbH
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

package tgo

import (
	"testing"
)

func TestMin(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(0, MinI(1, 0))
	expect.Equal(0, MinI(0, 1))
	expect.Equal(-1, MinI(-1, 1))
	expect.Equal(-1, MinI(1, -1))
}

func TestMax(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(1, MaxI(1, 0))
	expect.Equal(1, MaxI(0, 1))
	expect.Equal(1, MaxI(-1, 1))
	expect.Equal(1, MaxI(1, -1))
}

func TestMin3(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(0, Min3I(0, 1, 2))
	expect.Equal(0, Min3I(0, 2, 1))
	expect.Equal(0, Min3I(2, 1, 0))
	expect.Equal(0, Min3I(2, 0, 1))
	expect.Equal(0, Min3I(1, 0, 2))
	expect.Equal(0, Min3I(1, 2, 0))

	expect.Equal(-1, Min3I(-1, 0, 1))
	expect.Equal(-1, Min3I(-1, 1, 0))
	expect.Equal(-1, Min3I(0, -1, 1))
	expect.Equal(-1, Min3I(0, 1, -1))
	expect.Equal(-1, Min3I(1, -1, 0))
	expect.Equal(-1, Min3I(1, 0, -1))
}

func TestMax3(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(2, Max3I(0, 1, 2))
	expect.Equal(2, Max3I(0, 2, 1))
	expect.Equal(2, Max3I(2, 1, 0))
	expect.Equal(2, Max3I(2, 0, 1))
	expect.Equal(2, Max3I(1, 0, 2))
	expect.Equal(2, Max3I(1, 2, 0))

	expect.Equal(1, Max3I(-1, 0, 1))
	expect.Equal(1, Max3I(-1, 1, 0))
	expect.Equal(1, Max3I(0, -1, 1))
	expect.Equal(1, Max3I(0, 1, -1))
	expect.Equal(1, Max3I(1, -1, 0))
	expect.Equal(1, Max3I(1, 0, -1))
}
