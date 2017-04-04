// Copyright 2015-2017 trivago GmbH
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

	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/ttesting"
)

type mockConsumer struct {
	SimpleConsumer
}

func (mc *mockConsumer) Consume(wg *sync.WaitGroup) {
	// consumes something
}

func getMockConsumer() SimpleConsumer {
	return SimpleConsumer{
		control:  make(chan PluginControl),
		runState: NewPluginRunState(),
		Log:      tlog.NewLogScope("test"),
	}
}

func TestConsumerConfigure(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockC := getMockConsumer()

	pluginCfg := NewPluginConfig("mockConsumer", "mockConsumer")

	// Stream needs to be configured to avoid unknown class errors
	//Default stream name is plugin id
	registerMockStream("mockConsumer")

	err := mockC.Configure(NewPluginConfigReader(&pluginCfg))
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

	// accept timeroff by abs( 8 ms)
	tickThreshold := float64(10 * time.Millisecond)
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

func TestConsumerFuse(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockC := getMockConsumer()

	conf := NewPluginConfig("mockConsumer", "mockConsumer")
	conf.Override("Fuse", "test")

	mockC.Configure(NewPluginConfigReader(&conf))
	expect.NotNil(mockC.fuse)

	burnedCallback := new(int32)
	activeCallback := new(int32)

	mockC.SetFuseBurnedCallback(func() { atomic.StoreInt32(burnedCallback, 1) })
	mockC.SetFuseActiveCallback(func() { atomic.StoreInt32(activeCallback, 1) })

	go mockC.ControlLoop()

	expect.False(mockC.fuse.IsBurned())
	expect.Equal(atomic.LoadInt32(burnedCallback), int32(0))
	expect.Equal(atomic.LoadInt32(activeCallback), int32(0))

	// Check manual fuse trigger

	atomic.StoreInt32(burnedCallback, 0)
	atomic.StoreInt32(activeCallback, 0)

	mockC.Control() <- PluginControlFuseBurn
	time.Sleep(10 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(burnedCallback), int32(1))
	expect.Equal(atomic.LoadInt32(activeCallback), int32(0))

	atomic.StoreInt32(burnedCallback, 0)
	mockC.Control() <- PluginControlFuseActive
	time.Sleep(10 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(burnedCallback), int32(0))
	expect.Equal(atomic.LoadInt32(activeCallback), int32(1))

	// Check automatic burn callback

	atomic.StoreInt32(burnedCallback, 0)
	atomic.StoreInt32(activeCallback, 0)

	mockC.fuse.Burn()
	//time.Sleep(tsync.SpinTimeSuspend + 100*time.Millisecond)
	time.Sleep(time.Second + 100*time.Millisecond)

	expect.Equal(atomic.LoadInt32(burnedCallback), int32(1))
	expect.Equal(atomic.LoadInt32(activeCallback), int32(0))

	// Check automatic activate callback

	atomic.StoreInt32(burnedCallback, 0)
	mockC.fuse.Activate()
	time.Sleep(10 * time.Millisecond)
	expect.Equal(atomic.LoadInt32(burnedCallback), int32(0))
	expect.Equal(atomic.LoadInt32(activeCallback), int32(1))
}
