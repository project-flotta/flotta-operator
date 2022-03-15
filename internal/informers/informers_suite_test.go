package informers_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInformers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Informers Spec")
}
