package bench_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

var _ = Describe("ParseByteSize", func() {
	DescribeTable("valid inputs",
		func(input string, expected int64) {
			result, err := bench.ParseByteSize(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("plain number", "1024", int64(1024)),
		Entry("zero", "0", int64(0)),
		Entry("bytes suffix", "512b", int64(512)),
		Entry("bytes upper", "512B", int64(512)),
		Entry("kilobytes short", "4k", int64(4*1024)),
		Entry("kilobytes long", "4kb", int64(4*1024)),
		Entry("kilobytes upper", "4KB", int64(4*1024)),
		Entry("megabytes short", "10m", int64(10*1024*1024)),
		Entry("megabytes long", "10mb", int64(10*1024*1024)),
		Entry("megabytes upper", "10MB", int64(10*1024*1024)),
		Entry("gigabytes short", "1g", int64(1024*1024*1024)),
		Entry("gigabytes long", "1gb", int64(1024*1024*1024)),
		Entry("gigabytes mixed case", "2Gb", int64(2*1024*1024*1024)),
		Entry("with spaces", "  100 mb  ", int64(100*1024*1024)),
	)

	DescribeTable("invalid inputs",
		func(input string) {
			_, err := bench.ParseByteSize(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("empty string", ""),
		Entry("only letters", "abc"),
		Entry("negative", "-1kb"),
		Entry("decimal", "1.5mb"),
		Entry("unknown unit", "10tb"),
	)
})
