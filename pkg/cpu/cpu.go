package cpu

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// Config holds CPU scenario configuration.
type Config struct {
	Threads  int // number of parallel OS threads to spawn
	PiDigits int // decimal digits of pi to compute per thread
}

// Scenario implements bench.Scenario for CPU load testing.
// Each Run invocation spawns Threads goroutines, each locked to its own OS
// thread, and drives each thread to full utilisation by computing pi to
// PiDigits decimal digits using the Chudnovsky algorithm.
type Scenario struct {
	config Config
}

// New creates a new CPU load test Scenario.
func New(config Config) *Scenario {
	return &Scenario{config: config}
}

// Name implements bench.Scenario.
func (s *Scenario) Name() string { return "cpu" }

// Run spawns s.config.Threads goroutines. Each goroutine is locked to its own
// OS thread for the duration of the computation so that the load is spread
// across physical cores without interference from the Go scheduler.
func (s *Scenario) Run(ctx context.Context) bench.Result {
	start := time.Now()

	var wg sync.WaitGroup
	for range s.config.Threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			ComputePi(s.config.PiDigits)
		}()
	}
	wg.Wait()

	return bench.Result{Duration: time.Since(start)}
}
