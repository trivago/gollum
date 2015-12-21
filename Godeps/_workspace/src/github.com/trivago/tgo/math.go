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

// MaxI returns the maximum out of two integers
func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Max3I returns the maximum out of three integers
func Max3I(a, b, c int) int {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

// MinI returns the minimum out of two integers
func MinI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Min3I returns the minimum out of three integers
func Min3I(a, b, c int) int {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}
