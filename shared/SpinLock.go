package shared

import (
	"runtime"
	"sync/atomic"
)

// SpinLock is a lightweight, non-recursive mutex implementation.
type SpinLock struct {
	state uint32
}

// TryLock tries to acquire the lock and returns true on success.
func (lock *SpinLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&lock.state, 0, 1)
}

// Lock calls TryLock until TryLock succeeds. If the lock cannot be attained
// after 1024 tries the lock sleeps for 10 microseconds to give other threads
// a chance.
func (lock *SpinLock) Lock() {
	spinCount := 0
	for !lock.TryLock() {
		spinCount++
		if spinCount == 1024 {
			runtime.Gosched()
			spinCount = 0
		}
	}
}

// Unlock resets a lock.
func (lock *SpinLock) Unlock() {
	atomic.SwapUint32(&lock.state, 0)
}
