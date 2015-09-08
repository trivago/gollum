// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/shared"
	"math"
	"sync"
	"testing"
	"time"
)

type mockConsumer struct {
	ConsumerBase
}

func (mc *mockConsumer) Consume(wg *sync.WaitGroup) {
	// consumes something
}

func getMockConsumer() ConsumerBase {
	return ConsumerBase{
		control:  make(chan PluginControl),
		streams:  make([]MappedStream, 5),
		runState: NewPluginRunState(),
	}
}

func TestConsumerConfigure(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()

	pluginCfg := getMockPluginConfig()

	mockStream := getMockStream()
	mockStreamID := GetStreamID("mockStream")
	StreamRegistry.Register(&mockStream, mockStreamID)

	pluginCfg.Stream = []string{"mockStream"}

	err := mockC.Configure(pluginCfg)
	expect.Nil(err)
}

func TestConsumerEnqueueCopy(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()

	dataToSend := "Consumer Enqueue Data"
	distribute := func(msg Message) {
		expect.Equal(dataToSend, msg.String())
	}

	mockStream := getMockStream()
	mockP := getMockProducer()
	mockStream.AddProducer(&mockP)
	mockStream.distribute = distribute
	mockStreamID := GetStreamID("mockStream")
	StreamRegistry.Register(&mockStream, mockStreamID)

	mockC.streams = []MappedStream{
		MappedStream{
			StreamID: mockStreamID,
			Stream:   &mockStream,
		},
	}

	mockC.EnqueueCopy([]byte(dataToSend), 2)

}

func TestConsumerStreams(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()

	mockStream := getMockStream()
	mockP := getMockProducer()
	mockStream.AddProducer(&mockP)
	mockStreamID := GetStreamID("mockStream")
	StreamRegistry.Register(&mockStream, mockStreamID)

	mockC.streams = []MappedStream{
		MappedStream{
			StreamID: mockStreamID,
			Stream:   &mockStream,
		},
	}

	ret := mockC.Streams()
	expect.Equal(1, len(ret))
}

func TestConsumerControl(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()
	mockC.control = make(chan PluginControl, 2)

	ctrlChan := mockC.Control()
	ctrlChan <- PluginControlRoll
	ctrlChan <- PluginControlStopConsumer

	expect.Equal(PluginControlRoll, <-mockC.control)
	expect.Equal(PluginControlStopConsumer, <-mockC.control)
}

// For completeness' sake. This is exactly the same as Producer's ticket loop
func TestConsumerTickerLoop(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()
	mockC.setState(PluginStateActive)
	// accept timeroff by abs( 8 ms)
	delta := float64(8 * time.Millisecond)
	var counter = 0
	tickerLoopTimeout := 20 * time.Millisecond
	var timeRecorded time.Time
	onTimeOut := func() {
		if counter > 3 {
			mockC.setState(PluginStateDead)
			return
		}
		//this was fired as soon as the ticker started. So ignore but save the time
		if counter == 0 {
			timeRecorded = time.Now()
			counter++
			return
		}
		diff := time.Now().Sub(timeRecorded)
		deltaDiff := math.Abs(float64(tickerLoopTimeout - diff))
		expect.True(deltaDiff < delta)
		timeRecorded = time.Now()
		counter++
		return
	}

	mockC.tickerLoop(tickerLoopTimeout, onTimeOut)
	time.Sleep(2 * time.Second)
	// in anycase, the callback has to be called atleast once
	expect.True(counter > 1)
}

// For completeness' sake. This is exactly the same as Producer's control loop
func TestConsumerControlLoop(t *testing.T) {
	expect := shared.NewExpect(t)
	mockC := getMockConsumer()

	var stop bool
	var roll bool
	mockC.SetStopCallback(func() {
		stop = true
	})

	mockC.SetRollCallback(func() {
		roll = true
	})

	controlLoop := func() {
		mockC.ControlLoop()
	}

	go controlLoop()
	// let the go routine start
	time.Sleep(50 * time.Millisecond)
	mockC.control <- PluginControlStopConsumer
	time.Sleep(50 * time.Millisecond)
	expect.True(stop)

	go controlLoop()
	time.Sleep(50 * time.Millisecond)
	mockC.control <- PluginControlRoll
	time.Sleep(50 * time.Millisecond)
	expect.True(roll)
}
