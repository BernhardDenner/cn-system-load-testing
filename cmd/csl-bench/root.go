package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:     "csl-bench",
	Short:   "A cloud native system load testing tool",
	Long:    `csl-bench runs CPU, memory, disk and network load tests inside Kubernetes
clusters and reports performance metrics as JSON to stdout.`,
	Version: version,
}

func init() {
	// Common flags available to all sub-commands.
	rootCmd.PersistentFlags().IntP("duration", "d", 300,
		"seconds to run the benchmark; 0 for an infinite run")
	rootCmd.PersistentFlags().IntP("interval", "i", 1,
		"seconds between metric reports")
	rootCmd.PersistentFlags().StringArrayP("module", "m", nil,
		"test module to run (cpu, memory, disk, network); may be repeated")

	// Prometheus metrics endpoint.
	rootCmd.PersistentFlags().Int("metrics_port", 9090,
		"port for the Prometheus /metrics endpoint; 0 to disable")

	// Module-specific flags — CPU.
	rootCmd.PersistentFlags().Int("cpu_num_threads", 1,
		"number of threads for the cpu module")

	// Module-specific flags — Memory.
	rootCmd.PersistentFlags().Int("memory_max_use_mb", 0,
		"maximum memory in MB for the memory module; 0 = auto-detect from cgroup or system RAM")

	// Module-specific flags — Disk IO.
	rootCmd.PersistentFlags().String("io_mode", "randomized_rw",
		"disk IO mode (txn_rw, sequential_rw, randomized_rw)")
	rootCmd.PersistentFlags().String("io_file_path", "/tmp/bench-data",
		"path to the data file for IO operations")
	rootCmd.PersistentFlags().Int("io_batch_size_kb", 4,
		"batch size in KB for disk IO")
	rootCmd.PersistentFlags().Int("io_file_size_mb", 1024,
		"maximum file size in MB for disk IO")

	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(baselineCmd)
}
