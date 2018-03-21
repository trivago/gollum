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
	"strconv"
	"strings"
)

var versionString string

// GetVersionString return a symantic version string
func GetVersionString() string {
	if len(versionString) == 0 {
		return "0.0.0-non_make_build"
	}
	return versionString
}

// GetVersionNumber return a symantic based version number
func GetVersionNumber() int64 {
	parts := strings.Split(GetVersionString(), ".")
	multiplier := int64(100 * 100) // major, minor, patch
	version := int64(0)
	for i, subVerString := range parts {
		if i == 3 {
			break // done
		}
		subVer, err := strconv.Atoi(subVerString)
		if err != nil {
			break // cannot parse
		}
		version += int64(subVer) * multiplier
		multiplier /= 100

	}
	return version
}
