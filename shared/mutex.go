package shared

import (
	"sync/atomic"
)

// Mutex is a lightweight, spinner based mutex implementation, extending the
// standard go mutex by the possibility to query the mutexe's state. This state
// is very volatile but can come in handy sometimes.
type Mutex struct {
	state    *int32
	priority SpinPriority
}

// NewMutex creates a new mutex with the given spin priority used during Lock.
func NewMutex(priority SpinPriority) *Mutex {
	return &Mutex{
		state:    new(int32),
		priority: priority,
	}
}

// Lock blocks (spins) until the lock becomes available
func (m *Mutex) Lock() {
	spin := NewSpinner(m.priority)
	for !atomic.CompareAndSwapInt32(m.state, 0, 1) {
		spin.Yield()
	}
}

// Unlock unblocks one routine waiting on lock.
func (m *Mutex) Unlock() {
	atomic.StoreInt32(m.state, 0)
}

// IsLocked returns the state of this mutex. The result of this function might
// change directly after call so it should only be used in situations where
// this fact is not considered problematic.
func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32(m.state) == 1
}
