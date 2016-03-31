package tsync

import (
	"github.com/trivago/tgo/terrors"
)

// LockedError is returned when an item has been encountered as locked
type LockedError terrors.SimpleError

func (err LockedError) Error() string {
	return err.Error()
}

// TimeoutError is returned when a function returned because of a timeout
type TimeoutError terrors.SimpleError

func (err TimeoutError) Error() string {
	return err.Error()
}
