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

package main

import (
	"fmt"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 5
	gollumPatchVer = 0
	gollumDevVer   = 0
)

// GetVersionString return a symantic version string
func GetVersionString() string {
	var versionStr string

	if gollumDevVer > 0 {
		versionStr = fmt.Sprintf("v%d.%d.%d.%d-dev", gollumMajorVer, gollumMinorVer, gollumPatchVer, gollumDevVer)
	} else {
		versionStr = fmt.Sprintf("v%d.%d.%d", gollumMajorVer, gollumMinorVer, gollumPatchVer)
	}

	return versionStr
}

// GetVersionNumber return a symantic based version number
func GetVersionNumber() int64 {
	//todo: why are different multiplicators here in use?
	if gollumDevVer > 0 {
		return gollumMajorVer*1000000 + gollumMinorVer*10000 + gollumPatchVer*100 + gollumDevVer
	}

	return gollumMajorVer*10000 + gollumMinorVer*100 + gollumPatchVer
}
