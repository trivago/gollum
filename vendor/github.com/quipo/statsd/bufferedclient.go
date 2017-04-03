package statsd

import (
	"log"
	"os"
	"time"

	"github.com/quipo/statsd/event"
)

// request to close the buffered statsd collector
type closeRequest struct {
	reply chan error
}

// StatsdBuffer is a client library to aggregate events in memory before
// flushing aggregates to StatsD, useful if the frequency of events is extremely high
// and sampling is not desirable
type StatsdBuffer struct {
	statsd        *StatsdClient
	flushInterval time.Duration
	eventChannel  chan event.Event
	events        map[string]event.Event
	closeChannel  chan closeRequest
	Logger        Logger
	Verbose       bool
}

// NewStatsdBuffer Factory
func NewStatsdBuffer(interval time.Duration, client *StatsdClient) *StatsdBuffer {
	sb := &StatsdBuffer{
		flushInterval: interval,
		statsd:        client,
		eventChannel:  make(chan event.Event, 100),
		events:        make(map[string]event.Event, 0),
		closeChannel:  make(chan closeRequest, 0),
		Logger:        log.New(os.Stdout, "[BufferedStatsdClient] ", log.Ldate|log.Ltime),
		Verbose:       true,
	}
	go sb.collector()
	return sb
}

// CreateSocket creates a UDP connection to a StatsD server
func (sb *StatsdBuffer) CreateSocket() error {
	return sb.statsd.CreateSocket()
}

// CreateTCPSocket creates a TCP connection to a StatsD server
func (sb *StatsdBuffer) CreateTCPSocket() error {
	return sb.statsd.CreateTCPSocket()
}

// Incr - Increment a counter metric. Often used to note a particular event
func (sb *StatsdBuffer) Incr(stat string, count int64) error {
	if 0 != count {
		sb.eventChannel <- &event.Increment{Name: stat, Value: count}
	}
	return nil
}

// Decr - Decrement a counter metric. Often used to note a particular event
func (sb *StatsdBuffer) Decr(stat string, count int64) error {
	if 0 != count {
		sb.eventChannel <- &event.Increment{Name: stat, Value: -count}
	}
	return nil
}

// Timing - Track a duration event
func (sb *StatsdBuffer) Timing(stat string, delta int64) error {
	sb.eventChannel <- event.NewTiming(stat, delta)
	return nil
}

// PrecisionTiming - Track a duration event
// the time delta has to be a duration
func (sb *StatsdBuffer) PrecisionTiming(stat string, delta time.Duration) error {
	sb.eventChannel <- event.NewPrecisionTiming(stat, delta)
	return nil
}

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again
func (sb *StatsdBuffer) Gauge(stat string, value int64) error {
	sb.eventChannel <- &event.Gauge{Name: stat, Value: value}
	return nil
}

// GaugeDelta records a delta from the previous value (as int64)
func (sb *StatsdBuffer) GaugeDelta(stat string, value int64) error {
	sb.eventChannel <- &event.GaugeDelta{Name: stat, Value: value}
	return nil
}

// FGauge is a Gauge working with float64 values
func (sb *StatsdBuffer) FGauge(stat string, value float64) error {
	sb.eventChannel <- &event.FGauge{Name: stat, Value: value}
	return nil
}

// FGaugeDelta records a delta from the previous value (as float64)
func (sb *StatsdBuffer) FGaugeDelta(stat string, value float64) error {
	sb.eventChannel <- &event.FGaugeDelta{Name: stat, Value: value}
	return nil
}

// Absolute - Send absolute-valued metric (not averaged/aggregated)
func (sb *StatsdBuffer) Absolute(stat string, value int64) error {
	sb.eventChannel <- &event.Absolute{Name: stat, Values: []int64{value}}
	return nil
}

// FAbsolute - Send absolute-valued metric (not averaged/aggregated)
func (sb *StatsdBuffer) FAbsolute(stat string, value float64) error {
	sb.eventChannel <- &event.FAbsolute{Name: stat, Values: []float64{value}}
	return nil
}

// Total - Send a metric that is continously increasing, e.g. read operations since boot
func (sb *StatsdBuffer) Total(stat string, value int64) error {
	sb.eventChannel <- &event.Total{Name: stat, Value: value}
	return nil
}

// avoid too many allocations by memoizing the "type|key" pair for an event
// @see https://gobyexample.com/closures
func initMemoisedKeyMap() func(typ string, key string) string {
	m := make(map[string]map[string]string)
	return func(typ string, key string) string {
		if _, ok := m[typ]; !ok {
			m[typ] = make(map[string]string)
		}
		if k, ok := m[typ][key]; !ok {
			m[typ][key] = typ + "|" + key
			return m[typ][key]
		} else {
			return k // memoized value
		}
	}
}

// handle flushes and updates in one single thread (instead of locking the events map)
func (sb *StatsdBuffer) collector() {
	// on a panic event, flush all the pending stats before panicking
	defer func(sb *StatsdBuffer) {
		if r := recover(); r != nil {
			sb.Logger.Println("Caught panic, flushing stats before throwing the panic again")
			sb.flush()
			panic(r)
		}
	}(sb)

	keyFor := initMemoisedKeyMap() // avoid allocations (https://gobyexample.com/closures)

	ticker := time.NewTicker(sb.flushInterval)

	for {
		select {
		case <-ticker.C:
			//sb.Logger.Println("Flushing stats")
			sb.flush()
		case e := <-sb.eventChannel:
			//sb.Logger.Println("Received ", e.String())
			// issue #28: unable to use Incr and PrecisionTiming with the same key (also fixed #27)
			k := keyFor(e.TypeString(), e.Key()) // avoid allocations
			if e2, ok := sb.events[k]; ok {
				//sb.Logger.Println("Updating existing event")
				e2.Update(e)
				sb.events[k] = e2
			} else {
				//sb.Logger.Println("Adding new event")
				sb.events[k] = e
			}
		case c := <-sb.closeChannel:
			if sb.Verbose {
				sb.Logger.Println("Asked to terminate. Flushing stats before returning.")
			}
			c.reply <- sb.flush()
			return
		}
	}
}

// Close sends a close event to the collector asking to stop & flush pending stats
// and closes the statsd client
func (sb *StatsdBuffer) Close() (err error) {
	// 1. send a close event to the collector
	req := closeRequest{reply: make(chan error, 0)}
	sb.closeChannel <- req
	// 2. wait for the collector to drain the queue and respond
	err = <-req.reply
	// 3. close the statsd client
	err2 := sb.statsd.Close()
	if err != nil {
		return err
	}
	return err2
}

// send the events to StatsD and reset them.
// This function is NOT thread-safe, so it must only be invoked synchronously
// from within the collector() goroutine
func (sb *StatsdBuffer) flush() (err error) {
	n := len(sb.events)
	if n == 0 {
		return nil
	}
	if err := sb.statsd.SendEvents(sb.events); err != nil {
		sb.Logger.Println(err)
		return err
	}
	sb.events = make(map[string]event.Event)

	return nil
}
