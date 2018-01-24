package statsd

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/quipo/statsd/event"
)

// Logger interface compatible with log.Logger
type Logger interface {
	Println(v ...interface{})
}

// UDPPayloadSize is the number of bytes to send at one go through the udp socket.
// SendEvents will try to pack as many events into one udp packet.
// Change this value as per network capabilities
// For example to change to 16KB
//  import "github.com/quipo/statsd"
//  func init() {
//   statsd.UDPPayloadSize = 16 * 1024
//  }
var UDPPayloadSize = 512

// Hostname is exported so clients can set it to something different than the default
var Hostname string

var errNotConnected = fmt.Errorf("cannot send stats, not connected to StatsD server")

// errors
var (
	ErrInvalidCount      = errors.New("count is less than 0")
	ErrInvalidSampleRate = errors.New("sample rate is larger than 1 or less then 0")
)

func init() {
	if host, err := os.Hostname(); nil == err {
		Hostname = host
	}
}

type socketType string

const (
	udpSocket socketType = "udp"
	tcpSocket socketType = "tcp"
)

// StatsdClient is a client library to send events to StatsD
type StatsdClient struct {
	conn     net.Conn
	addr     string
	prefix   string
	sockType socketType
	Logger   Logger
}

// NewStatsdClient - Factory
func NewStatsdClient(addr string, prefix string) *StatsdClient {
	// allow %HOST% in the prefix string
	prefix = strings.Replace(prefix, "%HOST%", Hostname, 1)
	return &StatsdClient{
		addr:   addr,
		prefix: prefix,
		Logger: log.New(os.Stdout, "[StatsdClient] ", log.Ldate|log.Ltime),
	}
}

// String returns the StatsD server address
func (c *StatsdClient) String() string {
	return c.addr
}

// CreateSocket creates a UDP connection to a StatsD server
func (c *StatsdClient) CreateSocket() error {
	conn, err := net.DialTimeout(string(udpSocket), c.addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.conn = conn
	c.sockType = udpSocket
	return nil
}

// CreateTCPSocket creates a TCP connection to a StatsD server
func (c *StatsdClient) CreateTCPSocket() error {
	conn, err := net.DialTimeout(string(tcpSocket), c.addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.conn = conn
	c.sockType = tcpSocket
	return nil
}

// Close the UDP connection
func (c *StatsdClient) Close() error {
	if nil == c.conn {
		return nil
	}
	return c.conn.Close()
}

// See statsd data types here: http://statsd.readthedocs.org/en/latest/types.html
// or also https://github.com/b/statsd_spec

// Incr - Increment a counter metric. Often used to note a particular event
func (c *StatsdClient) Incr(stat string, count int64) error {
	return c.IncrWithSampling(stat, count, 1)
}

// IncrWithSampling - Increment a counter metric with sampling between 0 and 1
func (c *StatsdClient) IncrWithSampling(stat string, count int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if err := checkCount(count); err != nil {
		return err
	}

	return c.send(stat, "%d|c", count, sampleRate)
}

// Decr - Decrement a counter metric. Often used to note a particular event
func (c *StatsdClient) Decr(stat string, count int64) error {
	return c.DecrWithSampling(stat, count, 1)
}

// DecrWithSampling - Decrement a counter metric with sampling between 0 and 1
func (c *StatsdClient) DecrWithSampling(stat string, count int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if err := checkCount(count); err != nil {
		return err
	}

	return c.send(stat, "%d|c", -count, sampleRate)
}

// Timing - Track a duration event
// the time delta must be given in milliseconds
func (c *StatsdClient) Timing(stat string, delta int64) error {
	return c.TimingWithSampling(stat, delta, 1)
}

// TimingWithSampling - Track a duration event
func (c *StatsdClient) TimingWithSampling(stat string, delta int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	return c.send(stat, "%d|ms", delta, sampleRate)
}

// PrecisionTiming - Track a duration event
// the time delta has to be a duration
func (c *StatsdClient) PrecisionTiming(stat string, delta time.Duration) error {
	return c.send(stat, "%.6f|ms", float64(delta)/float64(time.Millisecond), 1)
}

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again. If you specify
// delta to be true, that specifies that the gauge should be updated, not set. Due to the
// underlying protocol, you can't explicitly set a gauge to a negative number without
// first setting it to zero.
func (c *StatsdClient) Gauge(stat string, value int64) error {
	return c.GaugeWithSampling(stat, value, 1)
}

// GaugeWithSampling - Gauges are a constant data type.
func (c *StatsdClient) GaugeWithSampling(stat string, value int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if value < 0 {
		err := c.send(stat, "%d|g", 0, 1)
		if nil != err {
			return err
		}
	}

	return c.send(stat, "%d|g", value, sampleRate)
}

// GaugeDelta -- Send a change for a gauge
func (c *StatsdClient) GaugeDelta(stat string, value int64) error {
	// Gauge Deltas are always sent with a leading '+' or '-'. The '-' takes care of itself but the '+' must added by hand
	if value < 0 {
		return c.send(stat, "%d|g", value, 1)
	}
	return c.send(stat, "+%d|g", value, 1)
}

// FGauge -- Send a floating point value for a gauge
func (c *StatsdClient) FGauge(stat string, value float64) error {
	return c.FGaugeWithSampling(stat, value, 1)
}

// FGaugeWithSampling - Gauges are a constant data type.
func (c *StatsdClient) FGaugeWithSampling(stat string, value float64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil
	}

	if value < 0 {
		err := c.send(stat, "%d|g", 0, 1)
		if nil != err {
			return err
		}
	}

	return c.send(stat, "%g|g", value, sampleRate)
}

// FGaugeDelta -- Send a floating point change for a gauge
func (c *StatsdClient) FGaugeDelta(stat string, value float64) error {
	if value < 0 {
		return c.send(stat, "%g|g", value, 1)
	}
	return c.send(stat, "+%g|g", value, 1)
}

// Absolute - Send absolute-valued metric (not averaged/aggregated)
func (c *StatsdClient) Absolute(stat string, value int64) error {
	return c.send(stat, "%d|a", value, 1)
}

// FAbsolute - Send absolute-valued floating point metric (not averaged/aggregated)
func (c *StatsdClient) FAbsolute(stat string, value float64) error {
	return c.send(stat, "%g|a", value, 1)
}

// Total - Send a metric that is continously increasing, e.g. read operations since boot
func (c *StatsdClient) Total(stat string, value int64) error {
	return c.send(stat, "%d|t", value, 1)
}

// write a UDP packet with the statsd event
func (c *StatsdClient) send(stat string, format string, value interface{}, sampleRate float32) error {
	if c.conn == nil {
		return errNotConnected
	}

	stat = strings.Replace(stat, "%HOST%", Hostname, 1)
	metricString := c.prefix + stat + ":" + fmt.Sprintf(format, value)

	if sampleRate != 1 {
		metricString = fmt.Sprintf("%s|@%f", metricString, sampleRate)
	}

	// if sending tcp append a newline
	if c.sockType == tcpSocket {
		metricString += "\n"
	}

	_, err := fmt.Fprint(c.conn, metricString)
	return err
}

// SendEvent - Sends stats from an event object
func (c *StatsdClient) SendEvent(e event.Event) error {
	if c.conn == nil {
		return errNotConnected
	}
	for _, stat := range e.Stats() {
		//fmt.Printf("SENDING EVENT %s%s\n", c.prefix, strings.Replace(stat, "%HOST%", Hostname, 1))
		_, err := fmt.Fprintf(c.conn, "%s%s", c.prefix, strings.Replace(stat, "%HOST%", Hostname, 1))
		if nil != err {
			return err
		}
	}
	return nil
}

// SendEvents - Sends stats from all the event objects.
// Tries to bundle many together into one fmt.Fprintf based on UDPPayloadSize.
func (c *StatsdClient) SendEvents(events map[string]event.Event) error {
	if c.conn == nil {
		return errNotConnected
	}

	var n int
	var stats = make([]string, 0)

	for _, e := range events {
		for _, stat := range e.Stats() {

			stat = c.prefix + strings.Replace(stat, "%HOST%", Hostname, 1)
			_n := n + len(stat) + 1

			if _n > UDPPayloadSize {
				// with this last event, the UDP payload would be too big
				if _, err := fmt.Fprintf(c.conn, strings.Join(stats, "\n")+"\n"); err != nil {
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
		if _, err := fmt.Fprintf(c.conn, strings.Join(stats, "\n")+"\n"); err != nil {
			return err
		}
	}

	return nil
}

func checkCount(c int64) error {
	if c <= 0 {
		return ErrInvalidCount
	}

	return nil
}

func checkSampleRate(r float32) error {
	if r < 0 || r > 1 {
		return ErrInvalidSampleRate
	}

	return nil
}

func shouldFire(sampleRate float32) bool {
	if sampleRate == 1 {
		return true
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))

	return r.Float32() <= sampleRate
}
