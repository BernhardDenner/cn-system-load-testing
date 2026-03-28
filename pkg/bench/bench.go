package bench

import (
	"context"
	"time"
)

// Config holds the benchmark run configuration.
type Config struct {
	Concurrency int
	Duration    time.Duration
	RampUp      time.Duration
}

// Result holds the outcome of a single scenario execution.
type Result struct {
	Duration time.Duration
	Err      error
}

// Scenario defines the interface for a load test scenario.
type Scenario interface {
	Name() string
	Run(ctx context.Context) Result
}

// Report contains aggregated benchmark results.
type Report struct {
	Scenario   string
	TotalOps   int64
	ErrorCount int64
	Duration   time.Duration
	Throughput float64 // ops/sec
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
}

// Runner executes a Scenario under load according to a Config.
type Runner struct {
	config   Config
	scenario Scenario
}

// NewRunner creates a new Runner for the given scenario and config.
func NewRunner(scenario Scenario, config Config) *Runner {
	return &Runner{
		config:   config,
		scenario: scenario,
	}
}

// Run executes the benchmark and returns a Report.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	// TODO: implement concurrent worker pool and metrics collection
	return &Report{
		Scenario: r.scenario.Name(),
	}, nil
}
