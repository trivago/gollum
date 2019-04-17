package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type mockRouterMessageHelper struct {
	SimpleRouter
	messageEnqued   bool
	lastMessageData string
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
			Logger:    logrus.WithField("Scope", "testStreamLogScope"),
		},
	}
}

func TestGetTargetDataFunction(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := NewGetterFor("")

	expect.Equal(reflect.Func, reflect.TypeOf(resultFunc).Kind())
}

func TestGetTargetDataFromPayload(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := NewGetterFor("")
	msg := NewMessage(nil, []byte("message payload"), nil, 1)

	expect.Equal([]byte("message payload"), resultFunc(msg).([]byte))
}

func TestGetTargetDataFromMetadata(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := NewGetterFor("foo")
	msg := NewMessage(nil, []byte("message payload"), nil, 1)
	msg.GetMetadata().Set("foo", "foo content")

	expect.Equal("foo content", resultFunc(msg).(string))
}
