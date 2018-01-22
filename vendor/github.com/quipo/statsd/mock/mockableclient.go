package mock

//
// A mockable client allowing arbitrary functions to be called statsd.Statsd methods.
//
// This is particularly helpful in unit test scenarios where it is desired to simulate calls
// without actually writing to a network or filesystem.
//
// A default implementation is provided that records all calls and can be used for verification
// in unit tests.
//
// The default implementations of these methods are no-ops, so without any further configuration,
// &MockStatsdClient{} is equivalent to statsd.NoopClient. But utility methods are also provided that
// allow recording calls for the purposes of verification during unit testing.
//

import (
	"sync"
	"time"

	"github.com/quipo/statsd/event"
)

type statelessStatsdFunction func() error
type intMetricStatsdFunction func(string, int64) error
type floatMetricStatsdFunction func(string, float64) error
type durationMetricStatsdFunction func(string, time.Duration) error
type eventsStatsdFunction func(events map[string]event.Event) error

// MockStatsdClient at its simplest provides a layer of indirection so that
// arbitrary functions can be used as the targets of calls to the Statsd interface
//
// In its basic state, it will act like a noop client.
//
// There are also helper functions that will set functions for specific
// calls to record those events to caller-provided slices.
// This is particularly helpful for unit testing that needs to verify
// that certain metrics have been recorded by the system under test

type MockStatsdClient struct {
	CreateSocketFn    statelessStatsdFunction
	CreateTCPSocketFn statelessStatsdFunction
	CloseFn           statelessStatsdFunction
	IncrFn            intMetricStatsdFunction
	DecrFn            intMetricStatsdFunction
	TimingFn          intMetricStatsdFunction
	PrecisionTimingFn durationMetricStatsdFunction
	GaugeFn           intMetricStatsdFunction
	GaugeDeltaFn      intMetricStatsdFunction
	AbsoluteFn        intMetricStatsdFunction
	TotalFn           intMetricStatsdFunction

	FGaugeFn      floatMetricStatsdFunction
	FGaugeDeltaFn floatMetricStatsdFunction
	FAbsoluteFn   floatMetricStatsdFunction

	SendEventsFn eventsStatsdFunction
}

// Implement statsd interface

func (msc *MockStatsdClient) CreateSocket() error {
	if msc.CreateSocketFn == nil {
		return nil
	}
	return msc.CreateSocketFn()
}

func (msc *MockStatsdClient) CreateTCPSocket() error {
	if msc.CreateTCPSocketFn == nil {
		return nil
	}
	return msc.CreateTCPSocketFn()
}

func (msc *MockStatsdClient) Close() error {
	if msc.CloseFn == nil {
		return nil
	}
	return msc.CloseFn()
}

func (msc *MockStatsdClient) Incr(stat string, count int64) error {
	if msc.IncrFn == nil {
		return nil
	}
	return msc.IncrFn(stat, count)
}

func (msc *MockStatsdClient) Decr(stat string, count int64) error {
	if msc.DecrFn == nil {
		return nil
	}
	return msc.DecrFn(stat, count)
}

func (msc *MockStatsdClient) Timing(stat string, delta int64) error {
	if msc.TimingFn == nil {
		return nil
	}
	return msc.TimingFn(stat, delta)
}

func (msc *MockStatsdClient) PrecisionTiming(stat string, delta time.Duration) error {
	if msc.PrecisionTimingFn == nil {
		return nil
	}
	return msc.PrecisionTimingFn(stat, delta)
}

func (msc *MockStatsdClient) Gauge(stat string, value int64) error {
	if msc.GaugeFn == nil {
		return nil
	}
	return msc.GaugeFn(stat, value)
}

func (msc *MockStatsdClient) GaugeDelta(stat string, value int64) error {
	if msc.GaugeDeltaFn == nil {
		return nil
	}
	return msc.GaugeDeltaFn(stat, value)
}

func (msc *MockStatsdClient) Absolute(stat string, value int64) error {
	if msc.AbsoluteFn == nil {
		return nil
	}
	return msc.AbsoluteFn(stat, value)
}

func (msc *MockStatsdClient) Total(stat string, value int64) error {
	if msc.TotalFn == nil {
		return nil
	}
	return msc.TotalFn(stat, value)
}

func (msc *MockStatsdClient) FGauge(stat string, value float64) error {
	if msc.FGaugeFn == nil {
		return nil
	}
	return msc.FGaugeFn(stat, value)
}

func (msc *MockStatsdClient) FGaugeDelta(stat string, value float64) error {
	if msc.FGaugeDeltaFn == nil {
		return nil
	}
	return msc.FGaugeDeltaFn(stat, value)
}

func (msc *MockStatsdClient) FAbsolute(stat string, value float64) error {
	if msc.FAbsoluteFn == nil {
		return nil
	}
	return msc.FAbsoluteFn(stat, value)
}

func (msc *MockStatsdClient) SendEvents(events map[string]event.Event) error {
	if msc.SendEventsFn == nil {
		return nil
	}
	return msc.SendEventsFn(events)
}

// Mocking helpers that record seen events for verification during unit testing

type Int64Event struct {
	MetricName string
	EventValue int64
}

type Float64Event struct {
	MetricName string
	EventValue float64
}

type DurationEvent struct {
	MetricName string
	EventValue time.Duration
}

// UnvaluedEvents are useful for recording things like calls to Close() or CreateSocket()
type UnvaluedEvent struct {
}

// Fluent-style constructors for recording events to caller-provided slices
// This means that during unit tests, one can do something like
//
// incrEvents := []statsd.Int64Event
// decrEvents := []statsd.Int64Event
// statsdClient = &MockStatsdClient{}.RecordIncrEventsTo(&incrEvents).RecordDecrEventsTo(&decrEvents)
// ... Execute code under test that records metrics
// ... Verify that incrEvents and decrEvents have seen the expected events

func (msc *MockStatsdClient) RecordCreateSocketEventsTo(createSocketEvents *[]UnvaluedEvent) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.CreateSocketFn = func() error {
		recordUnvaluedEvent(eventLock, createSocketEvents)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordCreateTCPSocketEventsTo(createTcpSocketEvents *[]UnvaluedEvent) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.CreateTCPSocketFn = func() error {
		recordUnvaluedEvent(eventLock, createTcpSocketEvents)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordCloseEventsTo(closeEvents *[]UnvaluedEvent) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.CloseFn = func() error {
		recordUnvaluedEvent(eventLock, closeEvents)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordIncrEventsTo(incrEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.IncrFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, incrEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordDecrEventsTo(decrEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.DecrFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, decrEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordTimingEventsTo(timingEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.TimingFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, timingEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordPrecisionTimingEventsTo(timingEvents *[]DurationEvent) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.PrecisionTimingFn = func(metricName string, eventValue time.Duration) error {
		recordDurationEvent(eventLock, timingEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordGaugeEventsTo(gaugeEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.GaugeFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, gaugeEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordGaugeDeltaEventsTo(gaugeDeltaEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.GaugeDeltaFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, gaugeDeltaEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordAbsoluteEventsTo(absoluteEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.AbsoluteFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, absoluteEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordTotalEventsTo(totalEvents *[]Int64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.TotalFn = func(metricName string, eventValue int64) error {
		recordInt64Event(eventLock, totalEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordFGaugeEventsTo(fgaugeEvents *[]Float64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.FGaugeFn = func(metricName string, eventValue float64) error {
		recordFloat64Event(eventLock, fgaugeEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordFGaugeDeltaEventsTo(fgaugeDeltaEvents *[]Float64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.FGaugeDeltaFn = func(metricName string, eventValue float64) error {
		recordFloat64Event(eventLock, fgaugeDeltaEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func (msc *MockStatsdClient) RecordFAbsoluteEventsTo(fabsoluteEvents *[]Float64Event) *MockStatsdClient {
	eventLock := &sync.Mutex{}
	msc.FAbsoluteFn = func(metricName string, eventValue float64) error {
		recordFloat64Event(eventLock, fabsoluteEvents, metricName, eventValue)
		return nil
	}
	return msc
}

func recordDurationEvent(eventLock sync.Locker, events *[]DurationEvent, metricName string, eventValue time.Duration) {
	newEvent := DurationEvent{
		MetricName: metricName,
		EventValue: eventValue,
	}
	eventLock.Lock()
	defer eventLock.Unlock()
	*events = append(*events, newEvent)
}

func recordFloat64Event(eventLock sync.Locker, events *[]Float64Event, metricName string, eventValue float64) {
	newEvent := Float64Event{
		MetricName: metricName,
		EventValue: eventValue,
	}
	eventLock.Lock()
	defer eventLock.Unlock()
	*events = append(*events, newEvent)
}

func recordInt64Event(eventLock sync.Locker, events *[]Int64Event, metricName string, eventValue int64) {
	newEvent := Int64Event{
		MetricName: metricName,
		EventValue: eventValue,
	}
	eventLock.Lock()
	defer eventLock.Unlock()
	*events = append(*events, newEvent)
}

func recordUnvaluedEvent(eventLock sync.Locker, events *[]UnvaluedEvent) {
	eventLock.Lock()
	defer eventLock.Unlock()
	*events = append(*events, UnvaluedEvent{})
}
