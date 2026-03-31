package bench_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// fixedDurationScenario returns a result with a fixed duration.
type fixedDurationScenario struct {
	dur time.Duration
}

func (s *fixedDurationScenario) Name() string { return "fixed" }
func (s *fixedDurationScenario) Run(_ context.Context) bench.Result {
	time.Sleep(s.dur)
	return bench.Result{Duration: s.dur}
}

var _ = Describe("LoadFactoredScenario", func() {
	It("delegates Name to the inner scenario", func() {
		inner := &fixedDurationScenario{dur: time.Millisecond}
		s := bench.NewLoadFactoredScenario(inner, 1.0)
		Expect(s.Name()).To(Equal("fixed"))
	})

	It("never calls inner when factor is 0", func() {
		inner := &instantIOScenario{bytesRead: 1024, bytesWritten: 1024}
		s := bench.NewLoadFactoredScenario(inner, 0)

		start := time.Now()
		for range 100 {
			s.Run(context.Background())
		}
		Expect(time.Since(start)).To(BeNumerically("<", 50*time.Millisecond))
	})

	It("runs without delay when factor is 1.0", func() {
		inner := &fixedDurationScenario{dur: 5 * time.Millisecond}
		s := bench.NewLoadFactoredScenario(inner, 1.0)

		start := time.Now()
		s.Run(context.Background())
		elapsed := time.Since(start)

		// Should take approximately 5ms, not significantly more.
		Expect(elapsed).To(BeNumerically(">=", 5*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 50*time.Millisecond))
	})

	It("sleeps proportionally for factor 0.5 to achieve ~50% duty cycle", func() {
		// Each inner run sleeps 10ms; with factor=0.5 the wrapper adds another
		// 10ms sleep, so total per op ≈ 20ms.  10 ops → ~200ms total.
		inner := &fixedDurationScenario{dur: 10 * time.Millisecond}
		s := bench.NewLoadFactoredScenario(inner, 0.5)

		start := time.Now()
		for range 10 {
			s.Run(context.Background())
		}
		elapsed := time.Since(start)

		Expect(elapsed).To(BeNumerically(">=", 150*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 400*time.Millisecond))
	})

	It("respects context cancellation during sleep", func() {
		inner := &fixedDurationScenario{dur: 5 * time.Millisecond}
		s := bench.NewLoadFactoredScenario(inner, 0.01) // 1% → 99% sleep

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		s.Run(ctx)
		Expect(time.Since(start)).To(BeNumerically("<", 500*time.Millisecond))
	})
})
