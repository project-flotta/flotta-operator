package factory_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/factory"
)

const namespace = "some-ns"

var _ = Describe("Backend factory", func() {

	It("should create k8s factory", func() {
		// given
		logger := &zap.SugaredLogger{}
		eventRecorder := record.NewFakeRecorder(1)
		var c client.Client

		// when
		backend := factory.Create(namespace, c, logger, eventRecorder)

		// then
		Expect(backend).ToNot(BeNil())
	})
})
