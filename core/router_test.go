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
	mock := getMockRouter()

	mockConf := NewPluginConfig("", "mockRouter")
	mockConf.Override("Stream", streamName)
	mockConf.Override("Modulators", []interface{}{
		"core.mockFormatter",
	})

	reader := NewPluginConfigReader(&mockConf)
	if err := reader.Configure(&mock); err != nil {
		panic(err)
	}
	StreamRegistry.Register(&mock, mock.GetStreamID())
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

	mock := getMockRouter()
	reader := NewPluginConfigReader(&mockConf)
	err := reader.Configure(&mock)
	expect.Equal(nil, err)
}

func TestStreamRoute(t *testing.T) {
	// TODO
}

func TestRouteOriginalMessage(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mock := getMockRouterMessageHelper("testStream")

	mockConf := NewPluginConfig("", "mockRouter")
	mockConf.Override("Stream", "messageDropStream")

	reader := NewPluginConfigReader(&mockConf)
	reader.Configure(&mock)
	StreamRegistry.Register(&mock, mock.GetStreamID())

	msg := NewMessage(nil, []byte("foo"), nil, mock.GetStreamID())
	msg.FreezeOriginal()

	err := RouteOriginal(msg, msg.GetRouter())
	expect.NoError(err)

	expect.True(mock.messageEnqued)
	expect.Equal("foo", mock.lastMessageData)

}

func TestRouteOriginal(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// create router mock A
	mockA := getMockRouterMessageHelper("testStreamA")

	mockConfA := NewPluginConfig("mockA", "mockRouterA")
	mockConfA.Override("Stream", "messageDropStreamA")

	reader := NewPluginConfigReader(&mockConfA)
	reader.Configure(&mockA)
	StreamRegistry.Register(&mockA, mockA.GetStreamID())

	// create router mock B
	mockB := getMockRouterMessageHelper("testStreamB")

	mockConfB := NewPluginConfig("mockB", "mockRouterB")
	mockConfB.Override("Stream", "messageDropStreamB")

	reader = NewPluginConfigReader(&mockConfB)
	reader.Configure(&mockB)
	StreamRegistry.Register(&mockB, mockB.GetStreamID())

	// create message and test
	msg := NewMessage(nil, []byte("foo"), nil, mockA.GetStreamID())
	msg.FreezeOriginal()

	err := RouteOriginal(msg, &mockB)
	expect.NoError(err)

	expect.False(mockA.messageEnqued)
	expect.Equal("", mockA.lastMessageData)
	expect.True(mockB.messageEnqued)
	expect.Equal("foo", mockB.lastMessageData)

}
