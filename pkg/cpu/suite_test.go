package cpu_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCPU(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CPU Suite")
}
