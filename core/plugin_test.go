// Copyright 2015-2018 trivago N.V.
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
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockPlugin struct{}

func (m *mockPlugin) Configure(config PluginConfigReader) {
}

func TestPluginRunState(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pluginState := NewPluginRunState()

	expect.Equal(PluginStateInitializing, pluginState.GetState())

	pluginState.SetState(PluginStateWaiting)
	expect.Equal(PluginStateWaiting, pluginState.GetState())
	pluginState.SetState(PluginStateActive)
	expect.Equal(PluginStateActive, pluginState.GetState())
	pluginState.SetState(PluginStateStopping)
	expect.Equal(PluginStateStopping, pluginState.GetState())

	var wg sync.WaitGroup
	pluginState.SetWorkerWaitGroup(&wg)

	pluginState.AddWorker()
	pluginState.AddWorker()
	done := new(int32)

	go func() {
		pluginState.WorkerDone()
		pluginState.WorkerDone()
		wg.Wait()
		atomic.StoreInt32(done, 1)
	}()
	// timeout for go routine.
	time.Sleep(500 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(done), int32(1))

}

func TestNewPlugin(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// Check no type given
	config, _ := NewNestedPluginConfig("", tcontainer.NewMarshalMap())
	_, err := NewPluginWithConfig(config)
	expect.NotNil(err) // No type used

	// Check missing configure method
	type notPlugin struct {
	}
	TypeRegistry.Register(notPlugin{})

	config, _ = NewNestedPluginConfig("core.notPlugin", tcontainer.NewMarshalMap())
	_, err = NewPluginWithConfig(config)
	expect.NotNil(err)

	// Check valid type and config
	TypeRegistry.Register(mockPlugin{})
	_, err = NewPluginWithConfig(NewPluginConfig("mockPluginConfig", "core.mockPlugin"))
	expect.NoError(err)
}
