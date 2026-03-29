package bench

import (
	"context"
	"io"
	"time"
)

// ThrottledScenario wraps a Scenario and enforces maximum read and write
// byte rates.  It tracks cumulative bytes transferred and sleeps as needed
// so the long-term average does not exceed the configured limits.
// A limit of 0 means unlimited for that direction.
type ThrottledScenario struct {
	inner    Scenario
	readBPS  int64
	writeBPS int64

	startTime         time.Time
	started           bool
	totalBytesRead    int64
	totalBytesWritten int64
}

// NewThrottledScenario wraps inner with the given byte-per-second rate
// limits.  Pass 0 for either limit to leave that direction unrestricted.
func NewThrottledScenario(inner Scenario, readBPS, writeBPS int64) *ThrottledScenario {
	return &ThrottledScenario{
		inner:    inner,
		readBPS:  readBPS,
		writeBPS: writeBPS,
	}
}

// Name delegates to the wrapped scenario.
func (t *ThrottledScenario) Name() string { return t.inner.Name() }

// Run executes the inner scenario and then sleeps if necessary to stay
// within the configured rate limits.  The returned Result.Duration reflects
// only the inner operation time, not the throttle delay.
func (t *ThrottledScenario) Run(ctx context.Context) Result {
	if !t.started {
		t.startTime = time.Now()
		t.started = true
	}

	r := t.inner.Run(ctx)

	t.totalBytesRead += r.BytesRead
	t.totalBytesWritten += r.BytesWritten

	if t.readBPS > 0 {
		expected := time.Duration(float64(t.totalBytesRead) / float64(t.readBPS) * float64(time.Second))
		if delay := expected - time.Since(t.startTime); delay > 0 {
			sleepCtx(ctx, delay)
		}
	}
	if t.writeBPS > 0 {
		expected := time.Duration(float64(t.totalBytesWritten) / float64(t.writeBPS) * float64(time.Second))
		if delay := expected - time.Since(t.startTime); delay > 0 {
			sleepCtx(ctx, delay)
		}
	}

	return r
}

// Close delegates to the inner scenario if it implements io.Closer.
func (t *ThrottledScenario) Close() error {
	if c, ok := t.inner.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
