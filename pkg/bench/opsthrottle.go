package bench

import (
	"context"
	"io"
	"time"
)

// OpsThrottledScenario wraps a Scenario and enforces a maximum operations-per-
// second rate.  It tracks cumulative op count and sleeps after each run when
// the actual rate exceeds the target.
//
// If the system cannot sustain the target rate (e.g. due to CPU starvation),
// no sleep occurs and the actual ops/sec falls below the target — which is
// reflected in the baseline_met metric.
type OpsThrottledScenario struct {
	inner     Scenario
	targetOPS float64

	startTime time.Time
	started   bool
	totalOps  int64
}

// NewOpsThrottledScenario wraps inner with the given target ops per second.
// targetOPS must be > 0.
func NewOpsThrottledScenario(inner Scenario, targetOPS float64) *OpsThrottledScenario {
	return &OpsThrottledScenario{inner: inner, targetOPS: targetOPS}
}

// Name delegates to the wrapped scenario.
func (s *OpsThrottledScenario) Name() string { return s.inner.Name() }

// Run executes the inner scenario and sleeps if necessary to stay within the
// configured ops/sec limit.
func (s *OpsThrottledScenario) Run(ctx context.Context) Result {
	if !s.started {
		s.startTime = time.Now()
		s.started = true
	}

	r := s.inner.Run(ctx)
	s.totalOps++

	expected := time.Duration(float64(s.totalOps) / s.targetOPS * float64(time.Second))
	if delay := expected - time.Since(s.startTime); delay > 0 {
		sleepCtx(ctx, delay)
	}

	return r
}

// Close delegates to the inner scenario if it implements io.Closer.
func (s *OpsThrottledScenario) Close() error {
	if c, ok := s.inner.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
