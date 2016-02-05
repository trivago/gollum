// Copyright 2015-2016 trivago GmbH
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

import (
	"runtime"
	"time"
)

// Spinner is a helper struct for spinning loops.
type Spinner struct {
	count    uint32
	priority SpinPriority
}

// SpinPriority is used for Spinner priority enum values
type SpinPriority uint32

const (
	// SpinPrioritySuspend should be used for spinning loops that are expected
	// to wait for very long periods of time. The loop will sleep for 1 second
	// after each iteration.
	SpinPrioritySuspend = SpinPriority(1)

	// SpinPriorityLow should be used for spinning loops that are expected to
	// spin for a rather long time before being able to exit.
	// After 100 loops the caller waits for 200 milliseconds.
	SpinPriorityLow = SpinPriority(100)

	// SpinPriorityMedium should be used for spinning loops that are expected to
	// spin for a short amount of time before being able to exit.
	// After 500 loops the caller waits for 100 milliseconds.
	SpinPriorityMedium = SpinPriority(500)

	// SpinPriorityHigh should be used for spinning loops that are expected to
	// almost never spin.
	// After 1000 loops the caller waits for 10 milliseconds.
	SpinPriorityHigh = SpinPriority(1000)

	// SpinPriorityRealtime should be used for loops that should never wait and
	// always spin. This priority will increase CPU load
	SpinPriorityRealtime = SpinPriority(0xFFFFFFFF)

	// SpinTimeSuspend holds the sleep time for SpinPrioritySuspend
	SpinTimeSuspend = time.Second
	// SpinTimeLow holds the sleep time for SpinPriorityLow
	SpinTimeLow = 200 * time.Millisecond
	// SpinTimeMedium holds the sleep time for SpinPriorityMedium
	SpinTimeMedium = 100 * time.Millisecond
	// SpinTimeHigh holds the sleep time for SpinPriorityHigh
	SpinTimeHigh = 10 * time.Millisecond
)

// NewSpinner creates a new helper for spinning loops
func NewSpinner(priority SpinPriority) Spinner {
	return Spinner{
		count:    0,
		priority: priority,
	}
}

// Yield should be called in spinning loops and will assure correct
// spin/wait/schedule behavior according to the set priority.
func (spin *Spinner) Yield() {
	if spin.count >= uint32(spin.priority) {
		spin.count = 0
		switch spin.priority {
		case SpinPrioritySuspend:
			time.Sleep(SpinTimeSuspend)
		case SpinPriorityLow:
			time.Sleep(SpinTimeLow)
		case SpinPriorityMedium:
			time.Sleep(SpinTimeMedium)
		case SpinPriorityHigh:
			time.Sleep(SpinTimeHigh)
		default:
			runtime.Gosched()
		}
	} else {
		spin.count++
		runtime.Gosched()
	}
}

// Reset sets the internal counter back to 0
func (spin *Spinner) Reset() {
	spin.count = 0
}
