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
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type mockConsumer struct {
	SimpleConsumer
}

func (mc *mockConsumer) Configure(conf PluginConfigReader) {
}

func (mc *mockConsumer) Consume(wg *sync.WaitGroup) {
	// consumes something
}

func getMockConsumer() mockConsumer {
	return mockConsumer{
		SimpleConsumer: SimpleConsumer{
			control:  make(chan PluginControl),
			runState: NewPluginRunState(),
			Logger:   logrus.WithField("Scope", "test"),
		},
	}
}

func TestConsumerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockC := getMockConsumer()

	pluginCfg := NewPluginConfig("mockConsumer", "mockConsumer")

	// Stream needs to be configured to avoid unknown class errors
	//Default stream name is plugin id
	registerMockRouter("mockConsumer")

	reader := NewPluginConfigReader(&pluginCfg)
	err := reader.Configure(&mockC)
	expect.NoError(err)
}

func TestConsumerControl(t *testing.T) {
	expect := ttesting.NewExpect(t)
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
	expect := ttesting.NewExpect(t)
	mockC := getMockConsumer()
	mockC.setState(PluginStateActive)

	// accept timeroff by abs( 15 ms)
	tickThreshold := float64(15 * time.Millisecond)
	tickerLoopTimeout := 20 * time.Millisecond
	lastTick := time.Now()
	counter := new(int32)

	onTimeOut := func() {
		defer func() {
			atomic.AddInt32(counter, 1)
			lastTick = time.Now()
		}()

		if atomic.LoadInt32(counter) > 3 {
			mockC.setState(PluginStateDead)
			return
		}

		if atomic.LoadInt32(counter) > 0 {
			deltaDiff := math.Abs(float64(tickerLoopTimeout - time.Since(lastTick)))
			expect.Less(deltaDiff, tickThreshold)
		}
	}

	mockC.tickerLoop(tickerLoopTimeout, onTimeOut)
	time.Sleep(2 * time.Second)
	// in anycase, the callback has to be called atleast once
	expect.Greater(atomic.LoadInt32(counter), int32(1))
}

// For completeness' sake. This is exactly the same as Producer's control loop
func TestConsumerControlLoop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockC := getMockConsumer()

	stop := new(int32)
	roll := new(int32)
	mockC.SetStopCallback(func() {
		atomic.StoreInt32(stop, 1)
	})

	mockC.SetRollCallback(func() {
		atomic.StoreInt32(roll, 1)
	})

	go mockC.ControlLoop()
	// let the go routine start
	time.Sleep(50 * time.Millisecond)
	mockC.control <- PluginControlStopConsumer
	time.Sleep(50 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(stop), int32(1))

	go mockC.ControlLoop()
	time.Sleep(50 * time.Millisecond)
	mockC.control <- PluginControlRoll
	time.Sleep(50 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(roll), int32(1))
}
