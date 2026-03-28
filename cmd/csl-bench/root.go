package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "csl-bench",
	Short: "A cloud native system load testing tool",
	Long: `csl-bench runs CPU, memory, disk and network load tests inside Kubernetes
clusters and reports performance metrics as JSON to stdout.`,
}

func init() {
	// Common flags available to all sub-commands.
	rootCmd.PersistentFlags().IntP("duration", "d", 300,
		"seconds to run the benchmark; 0 for an infinite run")
	rootCmd.PersistentFlags().IntP("interval", "i", 1,
		"seconds between metric reports")
	rootCmd.PersistentFlags().StringArrayP("module", "m", nil,
		"test module to run (cpu, memory, disk, network); may be repeated")

	// Module-specific flags.
	rootCmd.PersistentFlags().Int("cpu_num_threads", 1,
		"number of threads for the cpu module")

	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(baselineCmd)
}
