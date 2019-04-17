package main

import (
	"context"
	"net/http"
	"time"

	promMetrics "github.com/CrowdStrike/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
)

func startPrometheusMetricsService(address string) func() {
	srv := &http.Server{Addr: address}
	quit := make(chan struct{})
	prometheusRegistry := prometheus.NewRegistry()

	flushInterval := 3 * time.Second
	promClient := promMetrics.NewPrometheusProvider(core.MetricsRegistry, "gollum", "", prometheusRegistry, flushInterval)

	// Start updates
	go func() {
		for {
			select {
			case <-time.After(flushInterval):
				if err := promClient.UpdatePrometheusMetricsOnce(); err != nil {
					logrus.WithError(err).Warn("Error updating metrics")
				}
			case <-quit:
				return
			}
		}
	}()

	// Start http
	go func() {
		opts := promhttp.HandlerOpts{
			ErrorLog:      logrus.StandardLogger(),
			ErrorHandling: promhttp.ContinueOnError,
		}
		http.Handle("/prometheus", promhttp.HandlerFor(prometheusRegistry, opts))

		err := srv.ListenAndServe()
		if err != nil {
			logrus.WithError(err).Error("Failed to start metrics http server")
		}
	}()

	logrus.WithField("address", address).Info("Started metric service")

	// Return stop function
	return func() {
		close(quit)
		if err := srv.Shutdown(context.Background()); err != nil {
			logrus.WithError(err).Error("Failed to shutdown metrics http server")
		}
	}
}
