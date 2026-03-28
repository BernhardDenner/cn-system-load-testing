package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Run in baseline mode",
	Long: `Run load tests within defined usage limits and report whether
performance thresholds are met.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return fmt.Errorf("baseline mode is not yet implemented")
	},
}
