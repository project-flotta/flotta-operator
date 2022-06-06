package remote_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRemoteBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Remote backend Suite")
}
