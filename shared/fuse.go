package shared

import (
	"sync"
	"sync/atomic"
	"time"
)

// Fuse is a decentralized control mechanism that is ment to be used to manage
// the state of a certain resource. If the resource is not available the fuse is
// "burned" and a ticker function is provided that may be used to reactivate the
// fuse. Components depending on the resource guarded by the fuse may wait for
// the fuse until it becomes active again.
// Fuse is implementet in a threadsafe manner.
type Fuse struct {
	burnTimer     *time.Timer
	reactivate    *sync.Cond
	waitGuard     *sync.Mutex
	isStillBurned func() bool
	checkInterval time.Duration
	burned        *int32
}

// NewFuse creates a new Fuse and returns it.
// A new fuse is always active.
func NewFuse() *Fuse {
	return &Fuse{
		waitGuard:  new(sync.Mutex),
		reactivate: sync.NewCond(new(sync.Mutex)),
		burned:     new(int32),
	}
}

// Burn sets the fuse back to the "inactive" state.
// An already burned fuse cannot be burned again (call is ignored).
func (fuse *Fuse) Burn(checkInterval time.Duration, isStillBurned func() bool) {
	if !atomic.CompareAndSwapInt32(fuse.burned, 0, 1) {
		return // ### return, already burned ###
	}

	fuse.isStillBurned = isStillBurned
	fuse.checkInterval = checkInterval
	fuse.burnTimer = time.AfterFunc(checkInterval, fuse.checkBurnState)
}

// Activate sets the fuse back to the "running" state.
// An already active fuse cannot be activated again (call is ignored).
func (fuse *Fuse) Activate() {
	if !atomic.CompareAndSwapInt32(fuse.burned, 1, 0) {
		return // ### return, already active ###
	}

	// Use a mutex to avoid races with go routines calling Wait at this point
	fuse.waitGuard.Lock()
	defer fuse.waitGuard.Unlock()

	fuse.burnTimer.Stop()
	fuse.reactivate.Broadcast()
}

// Wait blocks until the fuse enters active state.
// Multiple go routines may wait on the same fuse.
func (fuse Fuse) Wait() {
	fuse.waitGuard.Lock()
	defer fuse.waitGuard.Unlock()

	if atomic.LoadInt32(fuse.burned) == 0 {
		return // ### return, not active ###
	}

	fuse.reactivate.Wait()
}

// IsBurned returns true if the fuse in the "inactive" state
func (fuse Fuse) IsBurned() bool {
	return atomic.LoadInt32(fuse.burned) == 1
}

func (fuse *Fuse) checkBurnState() {
	if atomic.LoadInt32(fuse.burned) == 0 {
		return // ### return, not active ###
	}

	if fuse.isStillBurned() {
		fuse.burnTimer = time.AfterFunc(fuse.checkInterval, fuse.checkBurnState)
	} else {
		fuse.Activate()
	}
}
