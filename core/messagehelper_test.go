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
			timeout:   timeout,
			streamID:  StreamRegistry.GetStreamID(streamName),
			Log:       tlog.NewLogScope("testStreamLogScope"),
		},
	}
}

func TestGetAppliedContentFunction(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentGetFunction("payload")

	expect.Equal(reflect.Func, reflect.TypeOf(resultFunc).Kind())
}

func TestGetAppliedContentFromPayload(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentGetFunction("payload")
	msg := NewMessage(nil, []byte("message payload"), 1)

	expect.Equal("message payload", string(resultFunc(msg)))
}

func TestGetAppliedContentFromMetadata(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := GetAppliedContentGetFunction("foo")
	msg := NewMessage(nil, []byte("message payload"), 1)
	msg.GetMetadata().SetValue("foo", []byte("foo content"))

	expect.Equal("foo content", string(resultFunc(msg)))
}
