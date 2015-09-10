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

	// Check reactivate of single wait

	start := time.Now()
	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	expect.False(fuse.IsBurned())
	expect.Less(int64(time.Since(start)), int64(150*time.Millisecond))

	// Check repeated burning

	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	expect.False(fuse.IsBurned())

	// Check reactivate of multiple waits

	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)

	time.Sleep(400 * time.Millisecond)
	expect.False(fuse.IsBurned())
}
