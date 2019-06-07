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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/trivago/tgo/ttesting"
)

func TestMessageTracer_Dump(t *testing.T) {
	expect := ttesting.NewExpect(t)

	metadata := NewMetadata()
	metadata.Set("foo", "bar")

	msg := NewMessage(messageTracerSource{}, []byte("test"), metadata, WildcardStreamID)
	out := sendMessageWithTrace(msg)

	expect.Contains(out, "TestMessageTracer")  // pluginID
	expect.Contains(out, `"test"`)             // payload
	expect.Contains(out, `"foo": "bar"`)       // metadata
	expect.Contains(out, "core.MessageTracer") // source

	DeactivateMessageTrace() // activate message tracing
}

func TestMessageTracer_DumpWithActiveTraceProducer(t *testing.T) {
	expect := ttesting.NewExpect(t)

	registerMockRouter(TraceInternalStream)
	//todo: init mock producer to get _TRACE_ messages

	msg := NewMessage(messageTracerSource{}, []byte("test"), nil, WildcardStreamID)
	out := sendMessageWithTrace(msg)

	expect.False(strings.Contains(out, "TestMessageTracer"))  // pluginID
	expect.False(strings.Contains(out, `"test"`))             // payload
	expect.False(strings.Contains(out, `"foo": "bar"`))       // metadata
	expect.False(strings.Contains(out, "core.MessageTracer")) // source
}

func sendMessageWithTrace(msg *Message) string {
	ActivateMessageTrace()

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	MessageTrace(msg, "TestMessageTracer", "test")

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	DeactivateMessageTrace()

	return string(out)
}
