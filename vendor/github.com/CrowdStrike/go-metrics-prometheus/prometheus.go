package prometheusmetrics

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

// PrometheusConfig provides a container with config parameters for the
// Prometheus Exporter
type PrometheusConfig struct {
	namespace     string
	registry      metrics.Registry // Registry to be exported
	subsystem     string
	promRegistry  prometheus.Registerer //Prometheus registry
	flushInterval time.Duration         //interval to update prom metrics
	counters      map[string]prometheus.Counter
	gauges        map[string]prometheus.Gauge
	gaugeVecs     map[string]prometheus.GaugeVec
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewPrometheusProvider(r metrics.Registry, namespace string, subsystem string, promReg prometheus.Registerer, flushInterval time.Duration) *PrometheusConfig {
	return &PrometheusConfig{
		namespace:     namespace,
		subsystem:     subsystem,
		registry:      r,
		promRegistry:  promReg,
		flushInterval: flushInterval,
		counters:      make(map[string]prometheus.Counter),
		gauges:        make(map[string]prometheus.Gauge),
		gaugeVecs:     make(map[string]prometheus.GaugeVec),
	}
}

// GetFlushInterval returns the time interval for metric update
func (c *PrometheusConfig) GetFlushInterval() time.Duration {
	return c.flushInterval
}

func (c *PrometheusConfig) flattenKey(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)
	return key
}

func (c *PrometheusConfig) counterFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	counter, ok := c.counters[key]
	if !ok {
		counter = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		})
		c.promRegistry.MustRegister(counter)
		c.counters[key] = counter
	}
	counter.Add(val)
}

func (c *PrometheusConfig) meterVec(name string, snap metrics.Meter) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	g, ok := c.gaugeVecs[key]
	if !ok {
		g = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		},
			[]string{
				"type",
			},
		)
		c.promRegistry.MustRegister(g)
		c.gaugeVecs[key] = g
	}

	g.WithLabelValues("count").Set(float64(snap.Count()))
	g.WithLabelValues("rate1").Set(snap.Rate1())
	g.WithLabelValues("rate5").Set(snap.Rate5())
	g.WithLabelValues("rate15").Set(snap.Rate15())
	g.WithLabelValues("rate_mean").Set(snap.RateMean())
}

func (c *PrometheusConfig) histogramVec(name string, snap metrics.Histogram) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	g, ok := c.gaugeVecs[key]
	if !ok {
		g = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		},
			[]string{
				"type",
			},
		)
		c.promRegistry.MustRegister(g)
		c.gaugeVecs[key] = g
	}
	g.WithLabelValues("count").Set(float64(snap.Count()))
	g.WithLabelValues("max").Set(float64(snap.Max()))
	g.WithLabelValues("min").Set(float64(snap.Min()))
	g.WithLabelValues("mean").Set(snap.Mean())
	g.WithLabelValues("stddev").Set(snap.StdDev())
	g.WithLabelValues("perc50").Set(snap.Percentile(float64(50)))
	g.WithLabelValues("perc75").Set(snap.Percentile(float64(75)))
	g.WithLabelValues("perc95").Set(snap.Percentile(float64(95)))
	g.WithLabelValues("perc99").Set(snap.Percentile(float64(99)))
	g.WithLabelValues("perc999").Set(snap.Percentile(float64(99.9)))
}

func (c *PrometheusConfig) gaugeFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	g, ok := c.gauges[key]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		})
		c.promRegistry.MustRegister(g)
		c.gauges[key] = g
	}
	g.Set(val)
}

// UpdatePrometheusMetrics via timeinterval
func (c *PrometheusConfig) UpdatePrometheusMetrics() {
	for range time.Tick(c.flushInterval) {
		if err := c.UpdatePrometheusMetricsOnce(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: updating prometheus Registry  %s\n", err)
		}
	}
}

// UpdatePrometheusMetricsOnce oneshot
func (c *PrometheusConfig) UpdatePrometheusMetricsOnce() error {
	c.registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			c.counterFromNameAndValue(name, float64(metric.Count()))
		case metrics.Gauge:
			//fmt.Fprintf(os.Stderr, "Gauge: %s %d\n", name, metric.Value())
			c.gaugeFromNameAndValue(name, float64(metric.Value()))
		case metrics.GaugeFloat64:
			//fmt.Fprintf(os.Stderr, "GaugeFloat64: %s %f\n", name, metric.Value())
			c.gaugeFromNameAndValue(name, float64(metric.Value()))
		case metrics.Histogram:
			snap := metric.Snapshot()
			c.histogramVec(name, snap)
		case metrics.Meter:
			snap := metric.Snapshot()
			c.meterVec(name, snap)
		case metrics.Timer:
			lastSample := metric.Snapshot().Rate1()
			c.gaugeFromNameAndValue(name, float64(lastSample))
		}
	})
	return nil
}
