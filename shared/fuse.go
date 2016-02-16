package shared

import (
	"sync"
	"sync/atomic"
)

// Fuse is a local circuit breaker implementation that is ment to be used to
// manage the state of a given resource between different threads of execution
// (consumer/producer). If the resource is not available the fuse is "burned".
// Components may now wait on that fuse and are woken as soon as the resource
// becomes available again (the fuse is "activated" again).
type Fuse struct {
	signal *sync.Cond
	burned *int32
}

// NewFuse creates a new Fuse and returns it.
// A new fuse is always active.
func NewFuse() *Fuse {
	return &Fuse{
		signal: sync.NewCond(new(sync.Mutex)),
		burned: new(int32),
	}
}

// IsBurned returns true if the fuse in the "inactive" state
func (fuse Fuse) IsBurned() bool {
	return atomic.LoadInt32(fuse.burned) == 1
}

// Burn sets the fuse back to the "inactive" state.
// An already burned fuse cannot be burned again (call is ignored).
func (fuse *Fuse) Burn() {
	atomic.StoreInt32(fuse.burned, 1)
}

// Activate sets the fuse back to the "running" state.
// An already active fuse cannot be activated again (call is ignored).
func (fuse *Fuse) Activate() {
	if atomic.CompareAndSwapInt32(fuse.burned, 1, 0) {
		fuse.signal.Broadcast()
	}
}

// Wait blocks until the fuse enters active state.
// Multiple go routines may wait on the same fuse.
func (fuse Fuse) Wait() {
	fuse.signal.L.Lock()
	defer fuse.signal.L.Unlock()
	if fuse.IsBurned() {
		fuse.signal.Wait()
	}
}
