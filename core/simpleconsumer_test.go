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
	"github.com/trivago/tgo/ttesting"
	"testing"
	"time"
)

func getSimpleConsumer(mockConf PluginConfig) (SimpleConsumer, error) {
	simpleConsumer := SimpleConsumer{
	//control:         make(chan PluginControl),
	//runState:        new(PluginRunState),
	//modulators:      ModulatorArray{},
	//Logger:          logrus.WithField("Scope", "test"),
	}

	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&simpleConsumer)

	return simpleConsumer, err
}

func TestSimpleConsumerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockConf := NewPluginConfig("mockSimpleConsumerConfigure", "mockSimpleConsumer")
	mockConf.Override("Streams", []string{"testBoundStream"})

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("testBoundStream")

	mockSimpleConsumer, err := getSimpleConsumer(mockConf)
	expect.NoError(err)

	expect.Equal(PluginStateInitializing, mockSimpleConsumer.runState.GetState())
}

func TestSimpleConsumerConfigureWithModulatorRoutines(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockConf := NewPluginConfig("mockSimpleConsumerConfigureWithModulatorRoutines", "mockSimpleConsumer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("ModulatorRoutines", 3)

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("testBoundStream")

	mockSimpleConsumer, err := getSimpleConsumer(mockConf)
	expect.NoError(err)

	expect.Equal(PluginStateInitializing, mockSimpleConsumer.runState.GetState())
}

func TestSimpleConsumerGetShutdownTimeout(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockConf := NewPluginConfig("mockSimpleConsumerGetShutdownTimeout", "mockSimpleConsumer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("ShutdownTimeoutMs", 100)

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("testBoundStream")

	mockSimpleConsumer, err := getSimpleConsumer(mockConf)
	expect.NoError(err)

	expect.Equal(time.Millisecond*100, mockSimpleConsumer.GetShutdownTimeout())
}

func TestSimpleConsumerStateMethods(t *testing.T) {
	expect := ttesting.NewExpect(t)

	mockConf := NewPluginConfig("mockSimpleConsumerStateMethods", "mockSimpleConsumer")
	mockConf.Override("Streams", []string{"testBoundStream"})
	mockConf.Override("ShutdownTimeoutMs", 100)

	// Router needs to be configured to avoid unknown class errors
	registerMockRouter("testBoundStream")

	mockSimpleConsumer, err := getSimpleConsumer(mockConf)
	expect.NoError(err)

	// PluginStateInitializing
	expect.False(mockSimpleConsumer.IsBlocked())
	expect.True(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.False(mockSimpleConsumer.IsStopping())

	// PluginStateWaiting
	mockSimpleConsumer.setState(PluginStateWaiting)
	expect.True(mockSimpleConsumer.IsBlocked())
	expect.True(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.False(mockSimpleConsumer.IsStopping())

	// PluginStateActive
	mockSimpleConsumer.setState(PluginStateActive)
	expect.False(mockSimpleConsumer.IsBlocked())
	expect.True(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.False(mockSimpleConsumer.IsStopping())

	// PluginStatePrepareStop
	mockSimpleConsumer.setState(PluginStatePrepareStop)
	expect.False(mockSimpleConsumer.IsBlocked())
	expect.True(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.True(mockSimpleConsumer.IsStopping())

	// PluginStateStopping
	mockSimpleConsumer.setState(PluginStateStopping)
	expect.False(mockSimpleConsumer.IsBlocked())
	expect.False(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.True(mockSimpleConsumer.IsStopping())

	// PluginStateDead
	mockSimpleConsumer.setState(PluginStateDead)
	expect.False(mockSimpleConsumer.IsBlocked())
	expect.False(mockSimpleConsumer.IsActive())
	expect.True(mockSimpleConsumer.IsActiveOrStopping())
	expect.True(mockSimpleConsumer.IsStopping())
}
