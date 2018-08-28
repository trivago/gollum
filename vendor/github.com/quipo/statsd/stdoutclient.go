package statsd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/quipo/statsd/event"
)

// StdoutClient implements a "no-op" statsd in case there is no statsd server
type StdoutClient struct {
	FD     *os.File
	prefix string
	Logger Logger
}

// NewStdoutClient - Factory
func NewStdoutClient(filename string, prefix string) *StdoutClient {
	var err error
	// allow %HOST% in the prefix string
	prefix = strings.Replace(prefix, "%HOST%", Hostname, 1)
	var fh *os.File
	if filename == "" {
		fh = os.Stdout
	} else {
		fh, err = os.OpenFile(filename, os.O_WRONLY, 0644)
		if nil != err {
			fmt.Printf("Cannot open file '%s' for stats output: %s\n", filename, err.Error())
		}
	}
	return &StdoutClient{
		FD:     fh,
		prefix: prefix,
		Logger: log.New(os.Stdout, "[StdoutClient] ", log.Ldate|log.Ltime),
	}
}

// CreateSocket does nothing
func (s *StdoutClient) CreateSocket() error {
	if s.FD == nil {
		s.FD = os.Stdout
	}
	return nil
}

// CreateTCPSocket does nothing
func (s *StdoutClient) CreateTCPSocket() error {
	if s.FD == nil {
		s.FD = os.Stdout
	}
	return nil
}

// Close does nothing
func (s *StdoutClient) Close() error {
	return nil
}

// Incr - Increment a counter metric. Often used to note a particular event
func (s *StdoutClient) Incr(stat string, count int64) error {
	if 0 != count {
		return s.send(stat, "%d|c", count)
	}
	return nil
}

// Decr - Decrement a counter metric. Often used to note a particular event
func (s *StdoutClient) Decr(stat string, count int64) error {
	if 0 != count {
		return s.send(stat, "%d|c", -count)
	}
	return nil
}

// Timing - Track a duration event
// the time delta must be given in milliseconds
func (s *StdoutClient) Timing(stat string, delta int64) error {
	return s.send(stat, "%d|ms", delta)
}

// PrecisionTiming - Track a duration event
// the time delta has to be a duration
func (s *StdoutClient) PrecisionTiming(stat string, delta time.Duration) error {
	return s.send(stat, "%.6f|ms", float64(delta)/float64(time.Millisecond))
}

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again. If you specify
// delta to be true, that specifies that the gauge should be updated, not set. Due to the
// underlying protocol, you can't explicitly set a gauge to a negative number without
// first setting it to zero.
func (s *StdoutClient) Gauge(stat string, value int64) error {
	if value < 0 {
		err := s.send(stat, "%d|g", 0)
		if nil != err {
			return err
		}
		return s.send(stat, "%d|g", value)
	}
	return s.send(stat, "%d|g", value)
}

// GaugeDelta -- Send a change for a gauge
func (s *StdoutClient) GaugeDelta(stat string, value int64) error {
	// Gauge Deltas are always sent with a leading '+' or '-'. The '-' takes care of itself but the '+' must added by hand
	if value < 0 {
		return s.send(stat, "%d|g", value)
	}
	return s.send(stat, "+%d|g", value)
}

// FGauge -- Send a floating point value for a gauge
func (s *StdoutClient) FGauge(stat string, value float64) error {
	if value < 0 {
		err := s.send(stat, "%d|g", 0)
		if nil != err {
			return err
		}
		return s.send(stat, "%g|g", value)
	}
	return s.send(stat, "%g|g", value)
}

// FGaugeDelta -- Send a floating point change for a gauge
func (s *StdoutClient) FGaugeDelta(stat string, value float64) error {
	if value < 0 {
		return s.send(stat, "%g|g", value)
	}
	return s.send(stat, "+%g|g", value)
}

// Absolute - Send absolute-valued metric (not averaged/aggregated)
func (s *StdoutClient) Absolute(stat string, value int64) error {
	return s.send(stat, "%d|a", value)
}

// FAbsolute - Send absolute-valued floating point metric (not averaged/aggregated)
func (s *StdoutClient) FAbsolute(stat string, value float64) error {
	return s.send(stat, "%g|a", value)
}

// Total - Send a metric that is continously increasing, e.g. read operations since boot
func (s *StdoutClient) Total(stat string, value int64) error {
	return s.send(stat, "%d|t", value)
}

// write a UDP packet with the statsd event
func (s *StdoutClient) send(stat string, format string, value interface{}) error {
	stat = strings.Replace(stat, "%HOST%", Hostname, 1)
	// if sending tcp append a newline
	format = fmt.Sprintf("%s%s:%s\n", s.prefix, stat, format)
	_, err := fmt.Fprintf(s.FD, format, value)
	return err
}

// SendEvent - Sends stats from an event object
func (s *StdoutClient) SendEvent(e event.Event) error {
	for _, stat := range e.Stats() {
		//fmt.Printf("SENDING EVENT %s%s\n", s.prefix, strings.Replace(stat, "%HOST%", Hostname, 1))
		_, err := fmt.Fprintf(s.FD, "%s%s", s.prefix, strings.Replace(stat, "%HOST%", Hostname, 1))
		if nil != err {
			return err
		}
	}
	return nil
}

// SendEvents - Sends stats from all the event objects.
// Tries to bundle many together into one fmt.Fprintf based on UDPPayloadSize.
func (s *StdoutClient) SendEvents(events map[string]event.Event) error {
	var n int
	var stats = make([]string, 0)

	for _, e := range events {
		for _, stat := range e.Stats() {

			stat = fmt.Sprintf("%s%s", s.prefix, strings.Replace(stat, "%HOST%", Hostname, 1))
			_n := n + len(stat) + 1

			if _n > UDPPayloadSize {
				// with this last event, the UDP payload would be too big
				if _, err := fmt.Fprintf(s.FD, strings.Join(stats, "\n")); err != nil {
					return err
				}
				// reset payload after flushing, and add the last event
				stats = []string{stat}
				n = len(stat)
				continue
			}

			// can fit more into the current payload
			n = _n
			stats = append(stats, stat)
		}
	}

	if len(stats) != 0 {
		if _, err := fmt.Fprintf(s.FD, strings.Join(stats, "\n")); err != nil {
			return err
		}
	}

	return nil
}
