package shared

import (
	"sync/atomic"
	"time"
)

type SpinLock struct {
	state uint32
}

func (lock *SpinLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&lock.state, 0, 1)
}

func (lock *SpinLock) Lock() {
	spinCount := 0
	for !lock.TryLock() {
		spinCount++
		if spinCount == 1024 {
			time.Sleep(time.Duration(10) * time.Microsecond)
			spinCount = 0
		}
	}
}

func (lock *SpinLock) Unlock() {
	atomic.SwapUint32(&lock.state, 0)
}
