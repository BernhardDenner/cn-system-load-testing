package memory_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/memory"
)

var _ = Describe("Scenario", func() {
	Describe("Name", func() {
		It("returns 'memory'", func() {
			s := memory.New(memory.Config{MaxUseMB: 10})
			Expect(s.Name()).To(Equal("memory"))
		})
	})

	Describe("Run", func() {
		It("completes successfully on first call", func() {
			s := memory.New(memory.Config{MaxUseMB: 10})
			result := s.Run(context.Background())
			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Duration).To(BeNumerically(">", 0))
		})

		It("reports bytes read and written", func() {
			s := memory.New(memory.Config{MaxUseMB: 10})
			result := s.Run(context.Background())
			Expect(result.BytesRead).To(Equal(int64(1 << 20)))
			Expect(result.BytesWritten).To(Equal(int64(1 << 20)))
		})

		It("survives more runs than pool capacity (eviction path)", func() {
			// 2 MB pool = 2 blocks; run 10 times to exercise eviction.
			s := memory.New(memory.Config{MaxUseMB: 2})
			for range 10 {
				result := s.Run(context.Background())
				Expect(result.Err).NotTo(HaveOccurred())
			}
		})

		It("works with the minimum pool size", func() {
			// MaxUseMB < 1 MB block → clamped to 1 block.
			s := memory.New(memory.Config{MaxUseMB: 0})
			result := s.Run(context.Background())
			Expect(result.Err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("availableMemoryBytes", func() {
	It("returns a positive value", func() {
		s := memory.New(memory.Config{MaxUseMB: 0})
		// Auto-detect path: scenario should initialise without error
		// and run successfully, implying a positive memory limit was detected.
		result := s.Run(context.Background())
		Expect(result.Err).NotTo(HaveOccurred())
	})
})
