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
	//	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type mockRouter struct {
	SimpleRouter
}

func (router *mockRouter) Configure(config PluginConfigReader) {
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
			id:        "testStream",
			filters:   FilterArray{},
			Producers: []Producer{},
			timeout:   timeout,
			streamID:  StreamRegistry.GetStreamID("testStream"),
			Logger:    logrus.WithField("Scope", "testStreamLogScope"),
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

	reader := NewPluginConfigReader(&mockConf)
	if err := reader.Configure(&mockRouter); err != nil {
		panic(err)
	}
	StreamRegistry.Register(&mockRouter, mockRouter.GetStreamID())
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
	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mockRouter)
	expect.Equal(nil, err)
}

func TestStreamRoute(t *testing.T) {
	// TODO
}

func TestRouteOriginalMessage(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockRouter := getMockRouterMessageHelper("testStream")

	mockConf := NewPluginConfig("", "mockRouter")
	mockConf.Override("Stream", "messageDropStream")

	reader := NewPluginConfigReader(&mockConf)
	reader.Configure(&mockRouter)
	StreamRegistry.Register(&mockRouter, mockRouter.GetStreamID())

	msg := NewMessage(nil, []byte("foo"), nil, mockRouter.GetStreamID())
	msg.FreezeOriginal()

	err := RouteOriginal(msg, msg.GetRouter())
	expect.NoError(err)

	expect.True(mockRouter.messageEnqued)
	expect.Equal("foo", mockRouter.lastMessageData)

}

func TestRouteOriginal(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// create router mock A
	mockRouterA := getMockRouterMessageHelper("testStreamA")

	mockConfA := NewPluginConfig("mockA", "mockRouterA")
	mockConfA.Override("Stream", "messageDropStreamA")

	reader := NewPluginConfigReader(&mockConfA)
	reader.Configure(&mockRouterA)
	StreamRegistry.Register(&mockRouterA, mockRouterA.GetStreamID())

	// create router mock B
	mockRouterB := getMockRouterMessageHelper("testStreamB")

	mockConfB := NewPluginConfig("mockB", "mockRouterB")
	mockConfB.Override("Stream", "messageDropStreamB")

	reader = NewPluginConfigReader(&mockConfB)
	reader.Configure(&mockRouterB)
	StreamRegistry.Register(&mockRouterB, mockRouterB.GetStreamID())

	// create message and test
	msg := NewMessage(nil, []byte("foo"), nil, mockRouterA.GetStreamID())
	msg.FreezeOriginal()

	err := RouteOriginal(msg, &mockRouterB)
	expect.NoError(err)

	expect.False(mockRouterA.messageEnqued)
	expect.Equal("", mockRouterA.lastMessageData)
	expect.True(mockRouterB.messageEnqued)
	expect.Equal("foo", mockRouterB.lastMessageData)

}
