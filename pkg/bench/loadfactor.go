package bench

import (
	"context"
	"io"
	"time"
)

// LoadFactoredScenario wraps a Scenario and controls its CPU or memory duty
// cycle.  A factor of 1.0 runs the inner scenario continuously; 0.5 sleeps
// for an equal duration after each run so the scenario is active half the
// time; 0.0 never runs the inner scenario.
//
// This is intended for background load generation where the load level must
// be bounded.  No baseline_met threshold is tracked — use OpsThrottledScenario
// if you need to measure whether a target ops/sec rate is being achieved.
type LoadFactoredScenario struct {
	inner  Scenario
	factor float64 // 0.0–1.0
}

// NewLoadFactoredScenario wraps inner with the given duty-cycle factor.
// factor must be in [0, 1]; values outside this range are clamped.
func NewLoadFactoredScenario(inner Scenario, factor float64) *LoadFactoredScenario {
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	return &LoadFactoredScenario{inner: inner, factor: factor}
}

// Name delegates to the wrapped scenario.
func (s *LoadFactoredScenario) Name() string { return s.inner.Name() }

// Run executes the inner scenario (unless factor == 0) and then sleeps for
// duration*(1-factor)/factor so that the long-term duty cycle matches factor.
func (s *LoadFactoredScenario) Run(ctx context.Context) Result {
	if s.factor == 0 {
		// Never run the inner scenario.
		return Result{}
	}

	r := s.inner.Run(ctx)

	if s.factor < 1 && r.Duration > 0 {
		// sleep = run_duration * (1 - factor) / factor
		sleepDur := time.Duration(float64(r.Duration) * (1 - s.factor) / s.factor)
		sleepCtx(ctx, sleepDur)
	}

	return r
}

// Close delegates to the inner scenario if it implements io.Closer.
func (s *LoadFactoredScenario) Close() error {
	if c, ok := s.inner.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
