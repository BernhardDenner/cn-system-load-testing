package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

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
	mode      string
	targets   []baselineTarget

	opsDesc              *prometheus.Desc
	errorsDesc           *prometheus.Desc
	latencyDesc          *prometheus.Desc
	bytesReadDesc        *prometheus.Desc
	bytesWrittenDesc     *prometheus.Desc
	baselineMetDesc      *prometheus.Desc
	targetReadBPSDesc    *prometheus.Desc
	targetWriteBPSDesc   *prometheus.Desc
	targetOpsPerSecDesc  *prometheus.Desc
}

func newMetricsCollector(scenarios []bench.Scenario, stats []moduleStats, mode string, targets []baselineTarget) *metricsCollector {
	return &metricsCollector{
		scenarios: scenarios,
		stats:     stats,
		mode:      mode,
		targets:   targets,
		opsDesc: prometheus.NewDesc(
			"csl_bench_ops_total",
			"Total number of operations performed.",
			[]string{"module", "mode"}, nil,
		),
		errorsDesc: prometheus.NewDesc(
			"csl_bench_errors_total",
			"Total number of failed operations.",
			[]string{"module", "mode"}, nil,
		),
		latencyDesc: prometheus.NewDesc(
			"csl_bench_latency_seconds_total",
			"Cumulative operation latency in seconds.",
			[]string{"module", "mode"}, nil,
		),
		bytesReadDesc: prometheus.NewDesc(
			"csl_bench_bytes_read_total",
			"Total bytes read.",
			[]string{"module", "mode"}, nil,
		),
		bytesWrittenDesc: prometheus.NewDesc(
			"csl_bench_bytes_written_total",
			"Total bytes written.",
			[]string{"module", "mode"}, nil,
		),
		baselineMetDesc: prometheus.NewDesc(
			"csl_bench_baseline_met",
			"Whether the baseline threshold is met (1) or not (0).",
			[]string{"module", "mode"}, nil,
		),
		targetReadBPSDesc: prometheus.NewDesc(
			"csl_bench_target_read_bps",
			"Configured target read bytes per second threshold.",
			[]string{"module", "mode"}, nil,
		),
		targetWriteBPSDesc: prometheus.NewDesc(
			"csl_bench_target_write_bps",
			"Configured target write bytes per second threshold.",
			[]string{"module", "mode"}, nil,
		),
		targetOpsPerSecDesc: prometheus.NewDesc(
			"csl_bench_target_ops_per_sec",
			"Configured target operations per second threshold.",
			[]string{"module", "mode"}, nil,
		),
	}
}

func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.opsDesc
	ch <- c.errorsDesc
	ch <- c.latencyDesc
	ch <- c.bytesReadDesc
	ch <- c.bytesWrittenDesc
	ch <- c.baselineMetDesc
	ch <- c.targetReadBPSDesc
	ch <- c.targetWriteBPSDesc
	ch <- c.targetOpsPerSecDesc
}

func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	for i, s := range c.scenarios {
		module := s.Name()
		snap := loadStats(&c.stats[i])
		tgt := targetFor(c.targets, i)

		ch <- prometheus.MustNewConstMetric(
			c.opsDesc, prometheus.CounterValue, float64(snap.ops), module, c.mode)
		ch <- prometheus.MustNewConstMetric(
			c.errorsDesc, prometheus.CounterValue, float64(snap.errors), module, c.mode)
		ch <- prometheus.MustNewConstMetric(
			c.latencyDesc, prometheus.CounterValue, float64(snap.latencyNs)/1e9, module, c.mode)
		ch <- prometheus.MustNewConstMetric(
			c.bytesReadDesc, prometheus.CounterValue, float64(snap.bytesRead), module, c.mode)
		ch <- prometheus.MustNewConstMetric(
			c.bytesWrittenDesc, prometheus.CounterValue, float64(snap.bytesWritten), module, c.mode)
		ch <- prometheus.MustNewConstMetric(
			c.baselineMetDesc, prometheus.GaugeValue, float64(atomic.LoadInt64(&c.stats[i].baselineMet)), module, c.mode)

		// Threshold gauges — only emitted when a target is configured.
		if tgt.readBPS > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.targetReadBPSDesc, prometheus.GaugeValue, float64(tgt.readBPS), module, c.mode)
		}
		if tgt.writeBPS > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.targetWriteBPSDesc, prometheus.GaugeValue, float64(tgt.writeBPS), module, c.mode)
		}
		if tgt.opsPerSec > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.targetOpsPerSecDesc, prometheus.GaugeValue, tgt.opsPerSec, module, c.mode)
		}
	}
}

// startMetricsServer registers the collector and starts an HTTP server
// serving /metrics on the given port.  The returned server can be shut down
// via Shutdown.
func startMetricsServer(port int, scenarios []bench.Scenario, stats []moduleStats, mode string, targets []baselineTarget) *http.Server {
	reg := prometheus.NewRegistry()
	reg.MustRegister(newMetricsCollector(scenarios, stats, mode, targets))

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
