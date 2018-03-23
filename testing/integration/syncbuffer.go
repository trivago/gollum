// +build integration

package integration

import (
	"bytes"
	"sync"
)

type syncBuffer struct {
	sync.Mutex
	data *bytes.Buffer
}

func newSyncBuffer(cap int) *syncBuffer {
	return &syncBuffer{
		data: bytes.NewBuffer(make([]byte, 0, cap)),
	}
}

func (b *syncBuffer) Write(data []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.data.Write(data)
}

func (b *syncBuffer) Read() string {
	b.Lock()
	defer b.Unlock()
	return b.data.String()
}
