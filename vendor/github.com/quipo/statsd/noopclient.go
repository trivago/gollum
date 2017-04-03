package statsd

//@author https://github.com/wyndhblb/statsd

import (
	"time"
)

// NoopClient implements a "no-op" statsd in case there is no statsd server
type NoopClient struct{}

// CreateSocket does nothing
func (s NoopClient) CreateSocket() error {
	return nil
}

// CreateTCPSocket does nothing
func (s NoopClient) CreateTCPSocket() error {
	return nil
}

// Close does nothing
func (s NoopClient) Close() error {
	return nil
}

// Incr does nothing
func (s NoopClient) Incr(stat string, count int64) error {
	return nil
}

// Decr does nothing
func (s NoopClient) Decr(stat string, count int64) error {
	return nil
}

// Timing does nothing
func (s NoopClient) Timing(stat string, count int64) error {
	return nil
}

// PrecisionTiming does nothing
func (s NoopClient) PrecisionTiming(stat string, delta time.Duration) error {
	return nil
}

// Gauge does nothing
func (s NoopClient) Gauge(stat string, value int64) error {
	return nil
}

// GaugeDelta does nothing
func (s NoopClient) GaugeDelta(stat string, value int64) error {
	return nil
}

// Absolute does nothing
func (s NoopClient) Absolute(stat string, value int64) error {
	return nil
}

// Total does nothing
func (s NoopClient) Total(stat string, value int64) error {
	return nil
}

// FGauge does nothing
func (s NoopClient) FGauge(stat string, value float64) error {
	return nil
}

// FGaugeDelta does nothing
func (s NoopClient) FGaugeDelta(stat string, value float64) error {
	return nil
}

// FAbsolute does nothing
func (s NoopClient) FAbsolute(stat string, value float64) error {
	return nil
}
