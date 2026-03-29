package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// metricsCollector implements prometheus.Collector by reading from the shared
// atomic counters on each scrape.  This avoids duplicating state — the
// run-loop atomics remain the single source of truth.
type metricsCollector struct {
	scenarios []bench.Scenario
	stats     []moduleStats

	opsDesc          *prometheus.Desc
	errorsDesc       *prometheus.Desc
	latencyDesc      *prometheus.Desc
	bytesReadDesc    *prometheus.Desc
	bytesWrittenDesc *prometheus.Desc
}

func newMetricsCollector(scenarios []bench.Scenario, stats []moduleStats) *metricsCollector {
	return &metricsCollector{
		scenarios: scenarios,
		stats:     stats,
		opsDesc: prometheus.NewDesc(
			"csl_bench_ops_total",
			"Total number of operations performed.",
			[]string{"module"}, nil,
		),
		errorsDesc: prometheus.NewDesc(
			"csl_bench_errors_total",
			"Total number of failed operations.",
			[]string{"module"}, nil,
		),
		latencyDesc: prometheus.NewDesc(
			"csl_bench_latency_seconds_total",
			"Cumulative operation latency in seconds.",
			[]string{"module"}, nil,
		),
		bytesReadDesc: prometheus.NewDesc(
			"csl_bench_bytes_read_total",
			"Total bytes read.",
			[]string{"module"}, nil,
		),
		bytesWrittenDesc: prometheus.NewDesc(
			"csl_bench_bytes_written_total",
			"Total bytes written.",
			[]string{"module"}, nil,
		),
	}
}

func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.opsDesc
	ch <- c.errorsDesc
	ch <- c.latencyDesc
	ch <- c.bytesReadDesc
	ch <- c.bytesWrittenDesc
}

func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	for i, s := range c.scenarios {
		module := s.Name()
		snap := loadStats(&c.stats[i])

		ch <- prometheus.MustNewConstMetric(
			c.opsDesc, prometheus.CounterValue, float64(snap.ops), module)
		ch <- prometheus.MustNewConstMetric(
			c.errorsDesc, prometheus.CounterValue, float64(snap.errors), module)
		ch <- prometheus.MustNewConstMetric(
			c.latencyDesc, prometheus.CounterValue, float64(snap.latencyNs)/1e9, module)
		ch <- prometheus.MustNewConstMetric(
			c.bytesReadDesc, prometheus.CounterValue, float64(snap.bytesRead), module)
		ch <- prometheus.MustNewConstMetric(
			c.bytesWrittenDesc, prometheus.CounterValue, float64(snap.bytesWritten), module)
	}
}

// startMetricsServer registers the collector and starts an HTTP server
// serving /metrics on the given port.  The returned server can be shut down
// via Shutdown.
func startMetricsServer(port int, scenarios []bench.Scenario, stats []moduleStats) *http.Server {
	reg := prometheus.NewRegistry()
	reg.MustRegister(newMetricsCollector(scenarios, stats))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go srv.ListenAndServe()
	return srv
}

// stopMetricsServer gracefully shuts down the HTTP server.
func stopMetricsServer(srv *http.Server) {
	if srv != nil {
		srv.Shutdown(context.Background())
	}
}
