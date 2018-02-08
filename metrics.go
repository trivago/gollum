package main

import (
	"net/http"
	"time"

	pm "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
)

func startPrometheusMetricsService(address string) func() {
	registry := prometheus.NewRegistry()
	srv := &http.Server{Addr: address}
	quit := make(chan struct{})

	// Start updates
	go func() {
		client := pm.NewPrometheusProvider(core.MetricsRegistry, "gollum", "", registry, 0)
		for {
			select {
			case <-time.After(time.Second):
				client.UpdatePrometheusMetricsOnce()
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
		http.Handle("/prometheus", promhttp.HandlerFor(registry, opts))

		err := srv.ListenAndServe()
		if err != nil {
			logrus.WithError(err).Error("Failed to start metrics http server")
		}
	}()

	logrus.WithField("address", address).Info("Started metric service")

	// Return stop function
	return func() {
		close(quit)
		if err := srv.Shutdown(nil); err != nil {
			logrus.WithError(err).Error("Failed to shutdown metrics http server")
		}
	}
}
