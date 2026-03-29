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
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Run in baseline mode",
	Long: `Run load tests within defined usage limits and report whether
performance thresholds are met.`,
	RunE: runBaseline,
}

func init() {
	// Baseline-specific flags for disk IO rate limiting.
	baselineCmd.Flags().Int64("io_read_bps", 0,
		"maximum read bytes per second for disk module; 0 = unlimited")
	baselineCmd.Flags().Int64("io_write_bps", 0,
		"maximum write bytes per second for disk module; 0 = unlimited")
}

func runBaseline(cmd *cobra.Command, _ []string) error {
	modules, _ := cmd.Flags().GetStringArray("module")
	durationSecs, _ := cmd.Flags().GetInt("duration")
	intervalSecs, _ := cmd.Flags().GetInt("interval")
	metricsPort, _ := cmd.Flags().GetInt("metrics_port")

	params := moduleParams{}
	params.cpuThreads, _ = cmd.Flags().GetInt("cpu_num_threads")
	params.memMaxUseMB, _ = cmd.Flags().GetInt("memory_max_use_mb")
	params.ioMode, _ = cmd.Flags().GetString("io_mode")
	params.ioFilePath, _ = cmd.Flags().GetString("io_file_path")
	params.ioBatchSizeKB, _ = cmd.Flags().GetInt("io_batch_size_kb")
	params.ioFileSizeMB, _ = cmd.Flags().GetInt("io_file_size_mb")

	ioReadBPS, _ := cmd.Flags().GetInt64("io_read_bps")
	ioWriteBPS, _ := cmd.Flags().GetInt64("io_write_bps")

	if len(modules) == 0 {
		return fmt.Errorf("at least one module must be specified with -m/--module")
	}

	scenarios, err := buildScenarios(modules, params)
	if err != nil {
		return err
	}

	// Wrap disk scenarios with rate limiting.
	if ioReadBPS > 0 || ioWriteBPS > 0 {
		for i, s := range scenarios {
			if s.Name() == "disk" {
				scenarios[i] = bench.NewThrottledScenario(s, ioReadBPS, ioWriteBPS)
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
	)
}
