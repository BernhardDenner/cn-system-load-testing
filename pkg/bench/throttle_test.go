package bench_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

// instantIOScenario returns immediately with fixed byte counts.
type instantIOScenario struct {
	bytesRead    int64
	bytesWritten int64
}

func (s *instantIOScenario) Name() string { return "instant-io" }
func (s *instantIOScenario) Run(_ context.Context) bench.Result {
	return bench.Result{
		Duration:     time.Microsecond,
		BytesRead:    s.bytesRead,
		BytesWritten: s.bytesWritten,
	}
}

var _ = Describe("ThrottledScenario", func() {
	It("delegates Name to the inner scenario", func() {
		inner := &instantIOScenario{bytesRead: 4096, bytesWritten: 4096}
		throttled := bench.NewThrottledScenario(inner, 0, 0)
		Expect(throttled.Name()).To(Equal("instant-io"))
	})

	It("does not throttle when limits are zero", func() {
		inner := &instantIOScenario{bytesRead: 4096, bytesWritten: 4096}
		throttled := bench.NewThrottledScenario(inner, 0, 0)

		start := time.Now()
		for range 100 {
			throttled.Run(context.Background())
		}
		Expect(time.Since(start)).To(BeNumerically("<", 100*time.Millisecond))
	})

	It("limits read throughput to the configured rate", func() {
		// 25 ops * 4096 bytes = 100 KB at 100 KB/s → ~1 second.
		inner := &instantIOScenario{bytesRead: 4096, bytesWritten: 0}
		throttled := bench.NewThrottledScenario(inner, 100*1024, 0)

		start := time.Now()
		for range 25 {
			throttled.Run(context.Background())
		}
		elapsed := time.Since(start)

		Expect(elapsed).To(BeNumerically(">=", 800*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 2*time.Second))
	})

	It("limits write throughput to the configured rate", func() {
		inner := &instantIOScenario{bytesRead: 0, bytesWritten: 4096}
		throttled := bench.NewThrottledScenario(inner, 0, 100*1024)

		start := time.Now()
		for range 25 {
			throttled.Run(context.Background())
		}
		elapsed := time.Since(start)

		Expect(elapsed).To(BeNumerically(">=", 800*time.Millisecond))
		Expect(elapsed).To(BeNumerically("<", 2*time.Second))
	})

	It("respects context cancellation during throttle sleep", func() {
		// Use a large read that would require a long sleep at 1 KB/s.
		inner := &instantIOScenario{bytesRead: 1 << 20, bytesWritten: 0}
		throttled := bench.NewThrottledScenario(inner, 1024, 0)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel shortly after the first Run triggers its long throttle sleep.
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		throttled.Run(ctx) // triggers ~1024s sleep, but ctx cancels after 50ms
		Expect(time.Since(start)).To(BeNumerically("<", 500*time.Millisecond))
	})
})
