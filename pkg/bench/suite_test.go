package bench_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBench(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bench Suite")
}
