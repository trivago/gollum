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

package shared

import "time"

// LowResolutionTimeNow is a cached time.Now() that gets updated every 10 to
// 100ms. This timer can be used in performance ciritcal situations where a
// nanosecond based timer is not required.
var LowResolutionTimeNow = time.Now()

func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		for {
			LowResolutionTimeNow = <-ticker.C
		}
	}()
}
