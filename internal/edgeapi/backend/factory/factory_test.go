package factory_test

import (
	"crypto/tls"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/factory"
)

const namespace = "some-ns"

var logger, _ = zap.NewDevelopment()

var _ = Describe("Backend factory", func() {

	It("should create k8s backend", func() {
		// given
		factory := factory.Factory{
			InitialDeviceNamespace: namespace,
			Logger:                 logger.Sugar(),
			Client:                 nil,
			EventRecorder:          record.NewFakeRecorder(1),
		}

		// when
		backend, _ := factory.Create("", time.Second)

		// then
		Expect(backend).ToNot(BeNil())
	})

	It("should create remote HTTP backend", func() {
		// given
		factory := factory.Factory{
			InitialDeviceNamespace: namespace,
			Logger:                 logger.Sugar(),
			Client:                 nil,
			EventRecorder:          record.NewFakeRecorder(1),
		}

		// when
		backend, _ := factory.Create("http://project-flotta.com", time.Second)

		// then
		Expect(backend).ToNot(BeNil())
	})

	It("should create remote HTTPS backend", func() {
		// given
		factory := factory.Factory{
			InitialDeviceNamespace: namespace,
			Logger:                 logger.Sugar(),
			Client:                 nil,
			EventRecorder:          record.NewFakeRecorder(1),
			TLSConfig:              &tls.Config{},
		}

		// when
		backend, _ := factory.Create("https://project-flotta.com", time.Second)

		// then
		Expect(backend).ToNot(BeNil())
	})
})
