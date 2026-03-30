package diskio_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/diskio"
)

var _ = Describe("ParseMode", func() {
	DescribeTable("accepts valid modes",
		func(input string, expected diskio.Mode) {
			m, err := diskio.ParseMode(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(Equal(expected))
		},
		Entry("txn_rw", "txn_rw", diskio.ModeTxnRW),
		Entry("sequential_rw", "sequential_rw", diskio.ModeSequentialRW),
		Entry("randomized_rw", "randomized_rw", diskio.ModeRandomizedRW),
	)

	It("rejects unknown modes", func() {
		_, err := diskio.ParseMode("bogus")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("bogus"))
	})
})

var _ = Describe("Scenario", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "diskio-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	newConfig := func(mode diskio.Mode) diskio.Config {
		return diskio.Config{
			Mode:      mode,
			FilePath:  filepath.Join(tmpDir, "bench-data"),
			BatchSize: 4 * 1024,       // 4 KB
			FileSize:  1 * 1024 * 1024, // 1 MB — small for fast tests
		}
	}

	Describe("Name", func() {
		It("returns 'disk'", func() {
			s := diskio.New(newConfig(diskio.ModeRandomizedRW))
			Expect(s.Name()).To(Equal("disk"))
		})
	})

	DescribeTable("Run completes successfully for each mode",
		func(mode diskio.Mode) {
			s := diskio.New(newConfig(mode))
			defer s.Close()

			result := s.Run(context.Background())
			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Duration).To(BeNumerically(">", 0))

			// Second call exercises the steady-state path (no init).
			result = s.Run(context.Background())
			Expect(result.Err).NotTo(HaveOccurred())
		},
		Entry("txn_rw", diskio.ModeTxnRW),
		Entry("sequential_rw", diskio.ModeSequentialRW),
		Entry("randomized_rw", diskio.ModeRandomizedRW),
	)

	It("creates the data file at the configured path with the expected size", func() {
		cfg := newConfig(diskio.ModeRandomizedRW)
		s := diskio.New(cfg)
		defer s.Close()

		_ = s.Run(context.Background())

		info, err := os.Stat(cfg.FilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Size()).To(Equal(cfg.FileSize))
	})

	It("removes the data file on Close", func() {
		cfg := newConfig(diskio.ModeRandomizedRW)
		s := diskio.New(cfg)

		_ = s.Run(context.Background())
		_, err := os.Stat(cfg.FilePath)
		Expect(err).NotTo(HaveOccurred(), "file should exist before Close")

		Expect(s.Close()).To(Succeed())
		_, err = os.Stat(cfg.FilePath)
		Expect(os.IsNotExist(err)).To(BeTrue(), "file should be removed after Close")
	})

	It("wraps sequential position around file boundary", func() {
		cfg := newConfig(diskio.ModeSequentialRW)
		cfg.FileSize = 1 * 1024 * 1024 // 1 MB / 4 KB = 256 batches
		s := diskio.New(cfg)
		defer s.Close()

		// Run more ops than the number of batches to exercise the wrap.
		for range 300 {
			result := s.Run(context.Background())
			Expect(result.Err).NotTo(HaveOccurred())
		}
	})
})
