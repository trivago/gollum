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

package tsync

import (
	"sync/atomic"
)

// Stack implements a simple, growing, lockfree stack.
type Stack struct {
	data   []interface{}
	growBy int
	top    *int32
	head   *int32
}

// NewStack creates a new stack with the given initial size.
// Keep in mind that size is also used to grow the stack, so a value > 1 is
// advisable.
func NewStack(size int) Stack {
	s := Stack{
		data:   make([]interface{}, size),
		growBy: size,
		top:    new(int32),
		head:   new(int32),
	}
	*s.top--
	*s.head--
	return s
}

// Pop retrieves the topmost element from the stack.
// A LimitError is returned when the stack is empty.
func (s *Stack) Pop() (interface{}, error) {
	for {
		top := atomic.LoadInt32(s.top)
		if top < 0 {
			return nil, LimitError{"Stack is empty"}
		}

		if atomic.CompareAndSwapInt32(s.head, top, top-1) {
			data := s.data[top]
			atomic.AddInt32(s.top, -1)
			return data, nil
		}
	}
}

// Push adds an element to the top of the stack.
// If the stack is full it is growed by its initial size and the element is added.
func (s *Stack) Push(v interface{}) {
	for {
		top := atomic.LoadInt32(s.top)
		if atomic.CompareAndSwapInt32(s.head, top, top+1) {
			if top+1 == int32(len(s.data)) {
				// Grow stack
				old := s.data
				s.data = make([]interface{}, len(s.data)+s.growBy)
				copy(s.data, old)
			}

			s.data[top+1] = v
			atomic.AddInt32(s.top, 1)
			return
		}
	}
}
