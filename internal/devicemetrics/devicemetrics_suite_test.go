package devicemetrics_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDeviceMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Device Metrics Suite")
}
