package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
	"github.com/BernhardDenner/cn-system-load-testing/pkg/cpu"
	"github.com/BernhardDenner/cn-system-load-testing/pkg/diskio"
	"github.com/BernhardDenner/cn-system-load-testing/pkg/memory"
)

// defaultCPUPiDigits controls how many digits of pi each CPU thread computes
// per operation.  10 000 digits gives a sub-millisecond workload on modern
// hardware while still producing measurable per-op latency.
const defaultCPUPiDigits = 10_000

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run in benchmark mode",
	Long:  `Run load tests without restrictions at full capacity.`,
	RunE:  runBenchmark,
}

// moduleParams bundles all per-module flag values so buildScenarios stays
// independent of cobra.
type moduleParams struct {
	cpuThreads      int
	memMaxUseBytes  int64
	ioMode          string
	ioFilePath      string
	ioBatchSize     int64
	ioFileSize      int64
}

func runBenchmark(cmd *cobra.Command, _ []string) error {
	modules, _ := cmd.Flags().GetStringArray("module")
	durationSecs, _ := cmd.Flags().GetInt("duration")
	intervalSecs, _ := cmd.Flags().GetInt("interval")
	metricsPort, _ := cmd.Flags().GetInt("metrics_port")

	params := moduleParams{}
	params.cpuThreads, _ = cmd.Flags().GetInt("cpu_num_threads")
	params.ioMode, _ = cmd.Flags().GetString("io_mode")
	params.ioFilePath, _ = cmd.Flags().GetString("io_file_path")

	if s, _ := cmd.Flags().GetString("memory_max_use"); s != "" {
		if v, err := bench.ParseByteSize(s); err != nil {
			return err
		} else {
			params.memMaxUseBytes = v
		}
	}
	if s, _ := cmd.Flags().GetString("io_batch_size"); s != "" {
		if v, err := bench.ParseByteSize(s); err != nil {
			return err
		} else {
			params.ioBatchSize = v
		}
	}
	if s, _ := cmd.Flags().GetString("io_file_size"); s != "" {
		if v, err := bench.ParseByteSize(s); err != nil {
			return err
		} else {
			params.ioFileSize = v
		}
	}

	if len(modules) == 0 {
		return fmt.Errorf("at least one module must be specified with -m/--module")
	}

	scenarios, err := buildScenarios(modules, params)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	defer closeScenarios(scenarios)

	return runLoop(
		ctx,
		scenarios,
		time.Duration(durationSecs)*time.Second,
		time.Duration(intervalSecs)*time.Second,
		metricsPort,
		bench.ModeBenchmark,
		nil,
	)
}

// buildScenarios creates Scenario instances for each requested module name.
// Duplicates are silently ignored.
func buildScenarios(modules []string, p moduleParams) ([]bench.Scenario, error) {
	seen := make(map[string]bool)
	var scenarios []bench.Scenario

	for _, m := range modules {
		if seen[m] {
			continue
		}
		seen[m] = true

		switch m {
		case "cpu":
			scenarios = append(scenarios, cpu.New(cpu.Config{
				Threads:  p.cpuThreads,
				PiDigits: defaultCPUPiDigits,
			}))
		case "disk":
			mode, err := diskio.ParseMode(p.ioMode)
			if err != nil {
				return nil, err
			}
			scenarios = append(scenarios, diskio.New(diskio.Config{
				Mode:      mode,
				FilePath:  p.ioFilePath,
				BatchSize: p.ioBatchSize,
				FileSize:  p.ioFileSize,
			}))
		case "memory":
			scenarios = append(scenarios, memory.New(memory.Config{
				MaxUseBytes: p.memMaxUseBytes,
			}))
		case "network":
			return nil, fmt.Errorf("module %q is not yet implemented", m)
		default:
			return nil, fmt.Errorf("unknown module %q: valid values are cpu, memory, disk, network", m)
		}
	}
	return scenarios, nil
}

// baselineTarget holds the configured byte-rate targets for one scenario.
// Used only in baseline mode; nil entries mean no targets.
type baselineTarget struct {
	readBPS  int64
	writeBPS int64
}

// moduleStats holds atomic counters for one running scenario.
type moduleStats struct {
	ops          int64
	errors       int64
	latencyNs    int64
	bytesRead    int64
	bytesWritten int64
	baselineMet  int64 // 1 = met, 0 = not met; used by Prometheus
}

// runLoop runs each scenario in its own goroutine for the given duration
// (or indefinitely when duration == 0) and prints a JSON Metrics line to
// stdout every interval.  A final summary line is printed after the run ends.
// targets is a per-scenario slice of baseline targets (may be nil for benchmark mode).
func runLoop(ctx context.Context, scenarios []bench.Scenario, duration, interval time.Duration, metricsPort int, mode string, targets []baselineTarget) error {
	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	stats := make([]moduleStats, len(scenarios))

	if metricsPort > 0 {
		srv := startMetricsServer(metricsPort, scenarios, stats, mode)
		defer stopMetricsServer(srv)
	}
	var wg sync.WaitGroup

	for i, s := range scenarios {
		wg.Add(1)
		go func(idx int, scenario bench.Scenario) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					r := scenario.Run(ctx)
					atomic.AddInt64(&stats[idx].ops, 1)
					atomic.AddInt64(&stats[idx].latencyNs, r.Duration.Nanoseconds())
					atomic.AddInt64(&stats[idx].bytesRead, r.BytesRead)
					atomic.AddInt64(&stats[idx].bytesWritten, r.BytesWritten)
					if r.Err != nil {
						atomic.AddInt64(&stats[idx].errors, 1)
					}
				}
			}
		}(i, s)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Previous snapshots — used to compute per-interval deltas.
	prev := make([]statsSnapshot, len(scenarios))
	lastTick := time.Now()

	for {
		select {
		case t := <-ticker.C:
			elapsed := t.Sub(lastTick)
			for i, s := range scenarios {
				snap := loadStats(&stats[i])
				tgt := targetFor(targets, i)
				m := emitMetrics(s.Name(), mode, bench.PhaseRunning, snap, prev[i], elapsed, tgt)
				if m.BaselineMet != nil {
					atomic.StoreInt64(&stats[i].baselineMet, int64(*m.BaselineMet))
				}
				prev[i] = snap
			}
			lastTick = t

		case <-ctx.Done():
			wg.Wait() // wait for in-flight scenario calls to finish
			elapsed := time.Since(lastTick)
			for i, s := range scenarios {
				snap := loadStats(&stats[i])
				tgt := targetFor(targets, i)
				m := emitMetrics(s.Name(), mode, bench.PhaseSummary, snap, prev[i], elapsed, tgt)
				if m.BaselineMet != nil {
					atomic.StoreInt64(&stats[i].baselineMet, int64(*m.BaselineMet))
				}
			}
			return nil
		}
	}
}

type statsSnapshot struct {
	ops          int64
	errors       int64
	latencyNs    int64
	bytesRead    int64
	bytesWritten int64
}

func loadStats(s *moduleStats) statsSnapshot {
	return statsSnapshot{
		ops:          atomic.LoadInt64(&s.ops),
		errors:       atomic.LoadInt64(&s.errors),
		latencyNs:    atomic.LoadInt64(&s.latencyNs),
		bytesRead:    atomic.LoadInt64(&s.bytesRead),
		bytesWritten: atomic.LoadInt64(&s.bytesWritten),
	}
}

func closeScenarios(scenarios []bench.Scenario) {
	for _, s := range scenarios {
		if c, ok := s.(io.Closer); ok {
			c.Close()
		}
	}
}

func targetFor(targets []baselineTarget, i int) baselineTarget {
	if i < len(targets) {
		return targets[i]
	}
	return baselineTarget{}
}

func emitMetrics(module, mode, phase string, snap, prev statsSnapshot, elapsed time.Duration, tgt baselineTarget) bench.Metrics {
	m := bench.NewMetrics(bench.MetricsInput{
		Mode:                 mode,
		Module:               module,
		Phase:                phase,
		Elapsed:              elapsed,
		TotalOps:             snap.ops,
		TotalErrors:          snap.errors,
		IntervalOps:          snap.ops - prev.ops,
		IntervalLatencyNs:    snap.latencyNs - prev.latencyNs,
		TotalBytesRead:       snap.bytesRead,
		TotalBytesWritten:    snap.bytesWritten,
		IntervalBytesRead:    snap.bytesRead - prev.bytesRead,
		IntervalBytesWritten: snap.bytesWritten - prev.bytesWritten,
		TargetReadBPS:        tgt.readBPS,
		TargetWriteBPS:       tgt.writeBPS,
	})
	data, _ := json.Marshal(m)
	fmt.Println(string(data))
	return m
}
