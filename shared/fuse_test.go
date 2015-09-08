package shared

import (
	"testing"
	"time"
)

func TestFuse(t *testing.T) {
	expect := NewExpect(t)
	fuse := NewFuse()

	expect.False(fuse.IsBurned())
	expect.NonBlocking(5*time.Millisecond, fuse.Wait)

	// Check reactivate (1)
	start := time.Now()
	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(time.Second, fuse.Activate)
	expect.NonBlocking(3*time.Second, fuse.Wait)
	expect.False(fuse.IsBurned())
	expect.Less(int64(time.Since(start)), int64(2*time.Second))
}
