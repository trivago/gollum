// Copyright 2015-2016 trivago GmbH
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

func getMockMessage(data string) Message {
	return Message{
		StreamID:     1,
		PrevStreamID: 2,
		Timestamp:    time.Now(),
		Sequence:     4,
		Data:         []byte(data),
	}
}

func TestMessageEnqueue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for Enqueue()"
	msg := getMockMessage(msgString)
	ch := make(chan Message)

	expect.Equal(MessageStateDiscard, msg.Enqueue(ch, -1))

	go func() {
		expect.Equal(MessageStateOk, msg.Enqueue(ch, 0))
	}()

	retMsg := (<-ch).String()
	expect.Equal(msgString, retMsg)

	retStatus := msg.Enqueue(ch, 10*time.Millisecond)
	expect.Equal(MessageStateTimeout, retStatus)

	go func() {
		expect.Equal(MessageStateOk, msg.Enqueue(ch, 1*time.Second))
	}()

	expect.Equal(msgString, (<-ch).String())
}

func TestMessageRoute(t *testing.T) {
	expect := ttesting.NewExpect(t)
	msgString := "Test for Route()"
	msg := getMockMessage(msgString)

	mockDistributer := func(msg Message) {
		expect.Equal(msgString, msg.String())
	}

	mockStream := StreamBase{}
	mockStream.Filter = &mockFilter{}
	mockStream.distribute = mockDistributer
	mockStream.formatters = []Formatter{mockFormatter{}}
	mockStream.AddProducer(&mockProducer{})
	StreamRegistry.Register(&mockStream, 1)

	msg.Route(1)

}
