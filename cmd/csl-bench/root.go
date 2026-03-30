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
	rootCmd.PersistentFlags().IntP("duration", "d", 0,
		"seconds to run the benchmark; 0 = run until cancelled (default)")
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
	rootCmd.PersistentFlags().String("memory_max_use", "0",
		"maximum memory for the memory module (e.g. 512mb, 2gb); 0 = auto-detect from cgroup or system RAM")

	// Module-specific flags — Disk IO.
	rootCmd.PersistentFlags().String("io_mode", "randomized_rw",
		"disk IO mode (txn_rw, sequential_rw, randomized_rw)")
	rootCmd.PersistentFlags().String("io_file_path", "/tmp/bench-data",
		"path to the data file for IO operations")
	rootCmd.PersistentFlags().String("io_batch_size", "4kb",
		"batch size for disk IO (e.g. 4kb, 1mb)")
	rootCmd.PersistentFlags().String("io_file_size", "1gb",
		"maximum data file size for disk IO (e.g. 512mb, 2gb)")

	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(baselineCmd)
}
