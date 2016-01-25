// Copyright 2015 trivago GmbH
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
	"fmt"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"sync"
	"testing"
	"time"
)

type mockPlugin struct{}

func (m *mockPlugin) Configure(config PluginConfig) error {
	if config.ID == "mockPluginConfig" {
		return nil
	}
	return fmt.Errorf("Plugin config id differs")
}

func TestPluginRunState(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pluginState := NewPluginRunState()

	expect.Equal(PluginStateDead, pluginState.GetState())

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
	done := false

	go func() {
		pluginState.WorkerDone()
		pluginState.WorkerDone()
		wg.Wait()
		done = true
	}()
	// timeout for go routine.
	time.Sleep(500 * time.Millisecond)
	expect.True(done)

}

func TestNewPlugin(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// Check no type given
	_, err := NewPlugin(NewNestedPluginConfig("", tcontainer.NewMarshalMap()))
	expect.NotNil(err) // No type used

	// Check inavlid type given
	type notPlugin struct {
	}
	TypeRegistry.Register(notPlugin{})
	_, err = NewPlugin(NewNestedPluginConfig("core.notPlugin", tcontainer.NewMarshalMap()))
	expect.NotNil(err)

	// Check valid type and config
	TypeRegistry.Register(mockPlugin{})
	_, err = NewPlugin(NewPluginConfig("", "mockPlugin"))
	expect.NoError(err)
}
