package statsd

import "time"

// Statsd is an interface to a StatsD client (buffered/unbuffered)
type Statsd interface {
	CreateSocket() error
	CreateTCPSocket() error
	Close() error
	Incr(stat string, count int64) error
	Decr(stat string, count int64) error
	Timing(stat string, delta int64) error
	PrecisionTiming(stat string, delta time.Duration) error
	Gauge(stat string, value int64) error
	GaugeDelta(stat string, value int64) error
	Absolute(stat string, value int64) error
	Total(stat string, value int64) error

	FGauge(stat string, value float64) error
	FGaugeDelta(stat string, value float64) error
	FAbsolute(stat string, value float64) error
}
