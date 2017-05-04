// Copyright 2015-2017 trivago GmbH
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
	//	"sync"
	"testing"
	"time"

	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/ttesting"
)

type mockRouter struct {
	SimpleRouter
}

func (router *mockRouter) Enqueue(msg *Message) error {
	return nil
}

func (router *mockRouter) Start() error {
	return nil
}

func getMockRouter() mockRouter {
	timeout := time.Second
	return mockRouter{
		SimpleRouter: SimpleRouter{
			id:         "testStream",
			modulators: ModulatorArray{},
			Producers:  []Producer{},
			Timeout:    &timeout,
			streamID:   StreamRegistry.GetStreamID("testStream"),
			Log:        tlog.NewLogScope("testStreamLogScope"),
		},
	}
}

func registerMockRouter(streamName string) {
	mockRouter := getMockRouter()

	mockConf := NewPluginConfig("", "mockRouter")
	mockConf.Override("Stream", streamName)
	mockConf.Override("Modulators", []interface{}{
		"core.mockFormatter",
	})

	mockRouter.Configure(NewPluginConfigReader(&mockConf))
	StreamRegistry.Register(&mockRouter, mockRouter.StreamID())
}

func TestRouterConfigureStream(t *testing.T) {
	expect := ttesting.NewExpect(t)
	TypeRegistry.Register(mockFormatter{})

	mockConf := NewPluginConfig("", "core.mockPlugin")
	mockConf.Override("Router", "testBoundStream")
	mockConf.Override("Modulators", []interface{}{
		"core.mockFormatter",
	})
	mockConf.Override("TimeoutMs", 100)

	mockRouter := getMockRouter()
	err := mockRouter.Configure(NewPluginConfigReader(&mockConf))
	expect.Equal(nil, err)
}

func TestStreamRoute(t *testing.T) {
	// TODO
}
