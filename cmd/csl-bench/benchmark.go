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
	cpuThreads    int
	ioMode        string
	ioFilePath    string
	ioBatchSizeKB int
	ioFileSizeMB  int
}

func runBenchmark(cmd *cobra.Command, _ []string) error {
	modules, _ := cmd.Flags().GetStringArray("module")
	durationSecs, _ := cmd.Flags().GetInt("duration")
	intervalSecs, _ := cmd.Flags().GetInt("interval")

	params := moduleParams{}
	params.cpuThreads, _ = cmd.Flags().GetInt("cpu_num_threads")
	params.ioMode, _ = cmd.Flags().GetString("io_mode")
	params.ioFilePath, _ = cmd.Flags().GetString("io_file_path")
	params.ioBatchSizeKB, _ = cmd.Flags().GetInt("io_batch_size_kb")
	params.ioFileSizeMB, _ = cmd.Flags().GetInt("io_file_size_mb")

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
				Mode:        mode,
				FilePath:    p.ioFilePath,
				BatchSizeKB: p.ioBatchSizeKB,
				FileSizeMB:  p.ioFileSizeMB,
			}))
		case "memory", "network":
			return nil, fmt.Errorf("module %q is not yet implemented", m)
		default:
			return nil, fmt.Errorf("unknown module %q: valid values are cpu, memory, disk, network", m)
		}
	}
	return scenarios, nil
}

// moduleStats holds atomic counters for one running scenario.
type moduleStats struct {
	ops       int64
	errors    int64
	latencyNs int64
}

// runLoop runs each scenario in its own goroutine for the given duration
// (or indefinitely when duration == 0) and prints a JSON Metrics line to
// stdout every interval.  A final summary line is printed after the run ends.
func runLoop(ctx context.Context, scenarios []bench.Scenario, duration, interval time.Duration) error {
	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	stats := make([]moduleStats, len(scenarios))
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
					if r.Err != nil {
						atomic.AddInt64(&stats[idx].errors, 1)
					}
				}
			}
		}(i, s)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Snapshot of counters at the previous tick — used to compute per-interval deltas.
	lastOps := make([]int64, len(scenarios))
	lastErrors := make([]int64, len(scenarios))
	lastLatNs := make([]int64, len(scenarios))
	lastTick := time.Now()
	start := lastTick

	for {
		select {
		case t := <-ticker.C:
			elapsed := t.Sub(lastTick)
			for i, s := range scenarios {
				snap := loadStats(&stats[i])
				printMetrics(s.Name(), bench.PhaseRunning,
					snap[0]-lastOps[i], snap[1]-lastErrors[i], snap[2]-lastLatNs[i], elapsed)
				lastOps[i], lastErrors[i], lastLatNs[i] = snap[0], snap[1], snap[2]
			}
			lastTick = t

		case <-ctx.Done():
			wg.Wait() // wait for in-flight scenario calls to finish
			total := time.Since(start)
			for i, s := range scenarios {
				snap := loadStats(&stats[i])
				printMetrics(s.Name(), bench.PhaseSummary, snap[0], snap[1], snap[2], total)
			}
			return nil
		}
	}
}

func loadStats(s *moduleStats) [3]int64 {
	return [3]int64{
		atomic.LoadInt64(&s.ops),
		atomic.LoadInt64(&s.errors),
		atomic.LoadInt64(&s.latencyNs),
	}
}

func closeScenarios(scenarios []bench.Scenario) {
	for _, s := range scenarios {
		if c, ok := s.(io.Closer); ok {
			c.Close()
		}
	}
}

func printMetrics(module, phase string, ops, errors, latNs int64, elapsed time.Duration) {
	m := bench.NewMetrics(module, phase, ops, errors, latNs, elapsed)
	data, _ := json.Marshal(m)
	fmt.Println(string(data))
}
