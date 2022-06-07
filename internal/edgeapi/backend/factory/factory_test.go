package factory_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"

	"github.com/project-flotta/flotta-operator/internal/edgeapi"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/factory"
)

const namespace = "some-ns"

var logger, _ = zap.NewDevelopment()

var _ = Describe("Backend factory", func() {
	var backendFactory factory.Factory

	BeforeEach(func() {
		backendFactory = factory.Factory{
			InitialDeviceNamespace: namespace,
			Logger:                 logger.Sugar(),
			Client:                 nil,
			EventRecorder:          record.NewFakeRecorder(1),
		}
	})

	DescribeTable("should create backend", func(config edgeapi.Config) {
		// when
		backend, err := backendFactory.Create(config)

		// then
		Expect(backend).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("k8s", edgeapi.Config{Backend: "crd"}),
		Entry("remote HTTP", edgeapi.Config{Backend: "remote", RemoteBackendURL: "http://project-flotta.com"}),
		Entry("remote HTTPS", edgeapi.Config{Backend: "remote", RemoteBackendURL: "https://project-flotta.com"}),
	)

	DescribeTable("should fail creating backend", func(config edgeapi.Config) {
		// when
		backend, err := backendFactory.Create(config)

		// then
		Expect(err).To(HaveOccurred())
		Expect(backend).To(BeNil())
	},
		Entry("illegal backend", edgeapi.Config{Backend: "foo"}),
		Entry("empty remote URL", edgeapi.Config{Backend: "remote"}),
		Entry("malformed remote URL", edgeapi.Config{Backend: "remote", RemoteBackendURL: "http://pr oject-flotta.com"}),
	)
})
