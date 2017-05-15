package core

import (
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/ttesting"
	"reflect"
	"testing"
	"time"
)

type mockRouterMessageHelper struct {
	SimpleRouter
	messageEnqued   bool
	lastMessageData string
}

func (router *mockRouterMessageHelper) init() {
	router.messageEnqued = false
	router.lastMessageData = ""
}

func (router *mockRouterMessageHelper) Enqueue(msg *Message) error {
	router.messageEnqued = true
	router.lastMessageData = msg.String()
	return nil
}

func (router *mockRouterMessageHelper) Start() error {
	return nil
}

func getMockRouterMessageHelper(streamName string) mockRouterMessageHelper {
	timeout := time.Second
	return mockRouterMessageHelper{
		SimpleRouter: SimpleRouter{
			id:        streamName,
			filters:   FilterArray{},
			Producers: []Producer{},
			Timeout:   &timeout,
			streamID:  StreamRegistry.GetStreamID(streamName),
			Log:       tlog.NewLogScope("testStreamLogScope"),
		},
	}
}

func TestGetAppliedContentFunction(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentFunction("payload")

	expect.Equal(reflect.Func, reflect.TypeOf(resultFunc).Kind())
}

func TestGetAppliedContentFromPayload(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentFunction("payload")
	msg := NewMessage(nil, []byte("message payload"), 1, 1)

	expect.Equal("message payload", string(resultFunc(msg)))
}

func TestGetAppliedContentFromMetaData(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentFunction("foo")
	msg := NewMessage(nil, []byte("message payload"), 1, 1)
	msg.MetaData().SetValue("foo", []byte("foo content"))

	expect.Equal("foo content", string(resultFunc(msg)))
}

func TestDropMessage(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockRouter := getMockRouterMessageHelper("testStream")

	mockConf := NewPluginConfig("", "mockRouter")
	mockConf.Override("Stream", "messageDropStream")

	mockRouter.Configure(NewPluginConfigReader(&mockConf))
	StreamRegistry.Register(&mockRouter, mockRouter.StreamID())

	msg := NewMessage(nil, []byte("foo"), 0, mockRouter.StreamID())

	err := RouteOriginal(msg, msg.GetRouter())
	expect.NoError(err)

	expect.True(mockRouter.messageEnqued)
	expect.Equal("foo", mockRouter.lastMessageData)

}

func TestDropMessageByRouter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// create router mock A
	mockRouterA := getMockRouterMessageHelper("testStreamA")

	mockConfA := NewPluginConfig("mockA", "mockRouterA")
	mockConfA.Override("Stream", "messageDropStreamA")

	mockRouterA.Configure(NewPluginConfigReader(&mockConfA))
	StreamRegistry.Register(&mockRouterA, mockRouterA.StreamID())

	// create router mock B
	mockRouterB := getMockRouterMessageHelper("testStreamB")

	mockConfB := NewPluginConfig("mockB", "mockRouterB")
	mockConfB.Override("Stream", "messageDropStreamB")

	mockRouterB.Configure(NewPluginConfigReader(&mockConfB))
	StreamRegistry.Register(&mockRouterB, mockRouterB.StreamID())

	// create message and test
	msg := NewMessage(nil, []byte("foo"), 0, mockRouterA.StreamID())

	err := RouteOriginal(msg, &mockRouterB)
	expect.NoError(err)

	expect.False(mockRouterA.messageEnqued)
	expect.Equal("", mockRouterA.lastMessageData)
	expect.True(mockRouterB.messageEnqued)
	expect.Equal("foo", mockRouterB.lastMessageData)

}
