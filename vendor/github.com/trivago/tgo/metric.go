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

package tgo

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// MetricProcessStart is the metric name storing the time when this process
	// has been started.
	MetricProcessStart = "ProcessStart"
	// MetricGoRoutines is the metric name storing the number of active go
	// routines.
	MetricGoRoutines = "GoRoutines"
)

// ProcessStartTime stores the time this process has started.
// This value is also stored in the metric MetricProcessStart
var ProcessStartTime time.Time

func init() {
	ProcessStartTime = time.Now()
	Metric.New(MetricProcessStart)
	Metric.New(MetricGoRoutines)
	Metric.Set(MetricProcessStart, ProcessStartTime.Unix())
}

type metrics struct {
	mutex *sync.Mutex
	store map[string]*int64
}

// Metric allows any part of gollum to store and/or modify metric values by
// name.
var Metric = metrics{new(sync.Mutex), make(map[string]*int64)}

// New creates a new metric under the given name with a value of 0
func (met *metrics) New(name string) {
	met.mutex.Lock()
	defer met.mutex.Unlock()
	if _, exists := met.store[name]; !exists {
		met.store[name] = new(int64)
	}
}

// Set sets a given metric to a given value. This operation is atomic.
func (met *metrics) Set(name string, value int64) {
	atomic.StoreInt64(met.store[name], value)
}

// SetI is Set for int values (conversion to int64)
func (met *metrics) SetI(name string, value int) {
	atomic.StoreInt64(met.store[name], int64(value))
}

// SetF is Set for float64 values (conversion to int64)
func (met *metrics) SetF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.StoreInt64(met.store[name], int64(rounded))
}

// Inc adds 1 to a given metric. This operation is atomic.
func (met *metrics) Inc(name string) {
	atomic.AddInt64(met.store[name], 1)
}

// Inc subtracts 1 from a given metric. This operation is atomic.
func (met *metrics) Dec(name string) {
	atomic.AddInt64(met.store[name], -1)
}

// Add adds a number to a given metric. This operation is atomic.
func (met *metrics) Add(name string, value int64) {
	atomic.AddInt64(met.store[name], value)
}

// AddI is Add for int values (conversion to int64)
func (met *metrics) AddI(name string, value int) {
	atomic.AddInt64(met.store[name], int64(value))
}

// AddF is Add for float64 values (conversion to int64)
func (met *metrics) AddF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.AddInt64(met.store[name], int64(rounded))
}

// Sub subtracts a number to a given metric. This operation is atomic.
func (met *metrics) Sub(name string, value int64) {
	atomic.AddInt64(met.store[name], -value)
}

// SubI is SubI for int values (conversion to int64)
func (met *metrics) SubI(name string, value int) {
	atomic.AddInt64(met.store[name], int64(-value))
}

// SubF is Sub for float64 values (conversion to int64)
func (met *metrics) SubF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	atomic.AddInt64(met.store[name], int64(-rounded))
}

// Get returns the value of a given metric. This operation is atomic.
// If the value does not exists error is non-nil and the returned value is 0.
func (met *metrics) Get(name string) (int64, error) {
	val, exists := met.store[name]
	if !exists {
		return 0, fmt.Errorf("Metric %s not found.", name)
	}
	return atomic.LoadInt64(val), nil
}

// UpdateSystemMetrics update all metrics that can be retrieved from the system
func (met *metrics) UpdateSystemMetrics() {
	met.SetI(MetricGoRoutines, runtime.NumGoroutine())
}

// Dump creates a JSON string from all stored metrics.
// This alos calls UpdateSystemMetrics
func (met *metrics) Dump() ([]byte, error) {
	met.UpdateSystemMetrics()
	return json.Marshal(Metric.store)
}
