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

package core

import (
	"github.com/trivago/tgo/tsync"
	"sync"
)

// fuseRegistry holds all fuses registered to the system.
type fuseRegistry struct {
	fuses     map[string]*tsync.Fuse
	fuseGuard *sync.Mutex
}

// FuseRegistry is the global instance of fuseRegistry used to store the
// all registered fuses.
var FuseRegistry = fuseRegistry{
	fuses:     make(map[string]*tsync.Fuse),
	fuseGuard: new(sync.Mutex),
}

// GetFuse returns a fuse object by name. This function will always return a
// valid fuse (creates fuses if they have not yet been created).
// This function is threadsafe.
func (registry *fuseRegistry) GetFuse(name string) *tsync.Fuse {
	registry.fuseGuard.Lock()
	defer registry.fuseGuard.Unlock()

	fuse, exists := registry.fuses[name]
	if !exists {
		fuse = tsync.NewFuse()
		registry.fuses[name] = fuse
	}
	return fuse
}

// ActivateAllFuses calls Activate on all registered fuses.
func (registry *fuseRegistry) ActivateAllFuses() {
	registry.fuseGuard.Lock()
	defer registry.fuseGuard.Unlock()

	for _, fuse := range registry.fuses {
		fuse.Activate()
	}
}
