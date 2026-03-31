package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
	"github.com/BernhardDenner/cn-system-load-testing/pkg/memory"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Run in baseline mode",
	Long: `Run load tests within defined usage limits and report whether
performance thresholds are met.`,
	RunE: runBaseline,
}

func init() {
	// Disk IO rate limiting.
	baselineCmd.Flags().String("io_read_bps", "0",
		"maximum read bytes per second for disk module (e.g. 50mb, 200kb); 0 = unlimited")
	baselineCmd.Flags().String("io_write_bps", "0",
		"maximum write bytes per second for disk module (e.g. 50mb, 200kb); 0 = unlimited")

	// CPU load control — mutually exclusive.
	baselineCmd.Flags().Float64("cpu_load_factor", 0,
		"CPU duty cycle for background load (0.0–1.0); 1.0 = run continuously, 0.5 = run 50%% of the time")
	baselineCmd.Flags().Float64("cpu_ops_per_sec", 0,
		"target CPU operations (pi calculations) per second; baseline_met=0 if rate falls below 98%% of target")

	// Memory load control.
	baselineCmd.Flags().Float64("memory_load_factor", 0,
		"fraction of available memory to allocate (0.0–1.0); 0 = use the memory_max_use value or auto-detect")
}

func runBaseline(cmd *cobra.Command, _ []string) error {
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

	var ioReadBPS, ioWriteBPS int64
	if s, _ := cmd.Flags().GetString("io_read_bps"); s != "" {
		if v, err := bench.ParseByteSize(s); err != nil {
			return err
		} else {
			ioReadBPS = v
		}
	}
	if s, _ := cmd.Flags().GetString("io_write_bps"); s != "" {
		if v, err := bench.ParseByteSize(s); err != nil {
			return err
		} else {
			ioWriteBPS = v
		}
	}

	cpuLoadFactor, _ := cmd.Flags().GetFloat64("cpu_load_factor")
	cpuOpsPerSec, _ := cmd.Flags().GetFloat64("cpu_ops_per_sec")
	memLoadFactor, _ := cmd.Flags().GetFloat64("memory_load_factor")

	if cpuLoadFactor != 0 && cpuOpsPerSec != 0 {
		return fmt.Errorf("--cpu_load_factor and --cpu_ops_per_sec are mutually exclusive")
	}

	// Apply memory load factor: override memMaxUseBytes with a fraction of
	// available memory.
	if memLoadFactor > 0 {
		params.memMaxUseBytes = int64(float64(memory.DetectMaxBytes()) * memLoadFactor)
	}

	if len(modules) == 0 {
		return fmt.Errorf("at least one module must be specified with -m/--module")
	}

	scenarios, err := buildScenarios(modules, params)
	if err != nil {
		return err
	}

	// Build per-scenario baseline targets and wrap scenarios as needed.
	targets := make([]baselineTarget, len(scenarios))
	for i, s := range scenarios {
		switch s.Name() {
		case "disk":
			targets[i] = baselineTarget{readBPS: ioReadBPS, writeBPS: ioWriteBPS}
			if ioReadBPS > 0 || ioWriteBPS > 0 {
				scenarios[i] = bench.NewThrottledScenario(s, ioReadBPS, ioWriteBPS)
			}
		case "cpu":
			switch {
			case cpuLoadFactor > 0:
				scenarios[i] = bench.NewLoadFactoredScenario(s, cpuLoadFactor)
			case cpuOpsPerSec > 0:
				targets[i] = baselineTarget{opsPerSec: cpuOpsPerSec}
				scenarios[i] = bench.NewOpsThrottledScenario(s, cpuOpsPerSec)
			}
		case "memory":
			if memLoadFactor > 0 {
				// Load factor already applied to memMaxUseBytes above;
				// no wrapper needed — the scenario runs at full rate with a
				// bounded memory pool.
			}
		}
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
		bench.ModeBaseline,
		targets,
	)
}
