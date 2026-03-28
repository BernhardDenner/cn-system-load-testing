package bench_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/BernhardDenner/cn-system-load-testing/pkg/bench"
)

type mockScenario struct {
	name string
}

func (m *mockScenario) Name() string { return m.name }
func (m *mockScenario) Run(_ context.Context) bench.Result {
	return bench.Result{Duration: time.Millisecond}
}

var _ = Describe("Runner", func() {
	var (
		scenario *mockScenario
		config   bench.Config
		runner   *bench.Runner
	)

	BeforeEach(func() {
		scenario = &mockScenario{name: "test-scenario"}
		config = bench.Config{
			Concurrency: 1,
			Duration:    time.Second,
		}
		runner = bench.NewRunner(scenario, config)
	})

	Describe("NewRunner", func() {
		It("creates a runner with the given scenario", func() {
			Expect(runner).NotTo(BeNil())
		})
	})

	Describe("Run", func() {
		It("returns a report with the scenario name", func() {
			report, err := runner.Run(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(report.Scenario).To(Equal("test-scenario"))
		})
	})
})
