package cpu_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/cpu"
)

var _ = Describe("Scenario", func() {
	Describe("Name", func() {
		It("returns 'cpu'", func() {
			s := cpu.New(cpu.Config{Threads: 1, PiDigits: 50})
			Expect(s.Name()).To(Equal("cpu"))
		})
	})

	Describe("Run", func() {
		It("completes successfully on a single thread", func() {
			s := cpu.New(cpu.Config{Threads: 1, PiDigits: 50})
			result := s.Run(context.Background())
			Expect(result.Err).To(BeNil())
			Expect(result.Duration).To(BeNumerically(">", 0))
		})

		It("completes successfully across multiple threads", func() {
			s := cpu.New(cpu.Config{Threads: 4, PiDigits: 50})
			result := s.Run(context.Background())
			Expect(result.Err).To(BeNil())
			Expect(result.Duration).To(BeNumerically(">", 0))
		})
	})
})

var _ = Describe("ComputePi", func() {
	// Known digits: 3.14159265358979323846264338327950288419716939937510…
	DescribeTable("produces correct digits",
		func(digits int, decimalPlaces int, expected string) {
			pi := cpu.ComputePi(digits)
			Expect(pi.Text('f', decimalPlaces)).To(Equal(expected))
		},
		Entry("50 digits, checked to 15 d.p.", 50, 15, "3.141592653589793"),
		Entry("100 digits, checked to 20 d.p.", 100, 20, "3.14159265358979323846"),
		Entry("200 digits, checked to 20 d.p.", 200, 20, "3.14159265358979323846"),
	)
})
