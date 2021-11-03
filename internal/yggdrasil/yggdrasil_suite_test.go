package yggdrasil_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestYggdrasil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Yggdrasil Suite")
}
