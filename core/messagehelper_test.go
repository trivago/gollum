package core

import (
	"github.com/trivago/tgo/ttesting"
	"testing"
	"reflect"
)

func TestGetAppliedContentFunction(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := getAppliedContentFunction("payload")

	expect.Equal(reflect.Func, reflect.TypeOf(resultFunc).Kind())
}

func TestGetAppliedContentFromPayload(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := getAppliedContentFunction("payload")
	msg := NewMessage(nil, []byte("message payload"), 1, 1)

	expect.Equal("message payload", string(resultFunc(msg)))
}

func TestGetAppliedContentFromMetaData(t *testing.T) {
	expect := ttesting.NewExpect(t)
	resultFunc := getAppliedContentFunction("meta:foo")
	msg := NewMessage(nil, []byte("message payload"), 1, 1)
	msg.MetaData().SetValue("foo", []byte("foo content"))

	expect.Equal("foo content", string(resultFunc(msg)))
}

func getAppliedContentFunction(applyTo string) GetAppliedContent {
	mockConf := NewPluginConfig("", "mockPlugin")
	mockConf.Override("ApplyTo", applyTo)

	conf := NewPluginConfigReader(&mockConf)

	return GetAppliedContentFunction(conf)
}