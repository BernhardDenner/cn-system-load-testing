package bench_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

var _ = Describe("OpsThrottledScenario", func() {
	It("delegates Name to the inner scenario", func() {
		inner := &instantIOScenario{}
		s := bench.NewOpsThrottledScenario(inner, 100)
		Expect(s.Name()).To(Equal("instant-io"))
	})

	It("limits ops to the configured rate", func() {
		// 20 ops at 10 ops/sec → ~2 seconds.
		inner := &instantIOScenario{}
		s := bench.NewOpsThrottledScenario(inner, 10)

		start := time.Now()
		for range 20 {
			s.Run(context.Background())
		}
		elapsed := time.Since(start)

		Expect(elapsed).To(BeNumerically(">=", 1600*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 3*time.Second))
	})

	It("does not sleep when ops are slower than the target rate", func() {
		// Each op takes 20ms; target is 100 ops/sec (10ms/op).
		// Actual rate (50/s) is already below target → no sleep.
		inner := &fixedDurationScenario{dur: 20 * time.Millisecond}
		s := bench.NewOpsThrottledScenario(inner, 100)

		start := time.Now()
		for range 5 {
			s.Run(context.Background())
		}
		elapsed := time.Since(start)

		// Should be ≈100ms (5×20ms) with no added sleep.
		Expect(elapsed).To(BeNumerically(">=", 80*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 400*time.Millisecond))
	})

	It("respects context cancellation during throttle sleep", func() {
		inner := &instantIOScenario{}
		s := bench.NewOpsThrottledScenario(inner, 1) // 1 op/sec → long sleep

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		s.Run(ctx) // triggers ~1s sleep, cancelled after 50ms
		Expect(time.Since(start)).To(BeNumerically("<", 500*time.Millisecond))
	})
})
