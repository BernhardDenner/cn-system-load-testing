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
	baselineCmd.Flags().String("io_read_bps", "0",
		"maximum read bytes per second for disk module (e.g. 50mb, 200kb); 0 = unlimited")
	baselineCmd.Flags().String("io_write_bps", "0",
		"maximum write bytes per second for disk module (e.g. 50mb, 200kb); 0 = unlimited")
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

	if len(modules) == 0 {
		return fmt.Errorf("at least one module must be specified with -m/--module")
	}

	scenarios, err := buildScenarios(modules, params)
	if err != nil {
		return err
	}

	// Build per-scenario baseline targets and wrap disk scenarios with rate limiting.
	targets := make([]baselineTarget, len(scenarios))
	for i, s := range scenarios {
		if s.Name() == "disk" {
			targets[i] = baselineTarget{readBPS: ioReadBPS, writeBPS: ioWriteBPS}
			if ioReadBPS > 0 || ioWriteBPS > 0 {
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
		bench.ModeBaseline,
		targets,
	)
}
