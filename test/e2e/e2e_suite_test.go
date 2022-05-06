package e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

// AfterFailed is a function that it's called on JustAfterEach to run a
// function if the test fail. For example, retrieving logs.
func AfterFailed(body func()) {
	JustAfterEach(func() {
		if CurrentSpecReport().Failed() {
			By("Running AfterFailed function")
			body()
		}
	})
}
