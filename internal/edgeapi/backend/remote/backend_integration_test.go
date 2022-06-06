package remote_test

import (
	"context"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/project-flotta/flotta-operator/backend/client"
	backendapi "github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/remote"
	"github.com/project-flotta/flotta-operator/models"
)

const (
	initialNamespace = "some-ns"
	namespace        = "ns"
	deviceID         = "deviceID"
)

var _ = Describe("Remote backend", func() {

	var httpServer *server
	var backend backendapi.EdgeDeviceBackend
	var logger *zap.Logger

	createBackend := func(httpServer *server) {
		serverURL, err := url.Parse("http://" + httpServer.address + client.DefaultBasePath)
		Expect(err).ToNot(HaveOccurred())

		config := client.Config{URL: serverURL}
		backendApi := client.New(config)
		backend = remote.NewBackend(initialNamespace, backendApi, 5*time.Second, logger.Sugar())
	}

	BeforeEach(func() {
		logger, _ = zap.NewDevelopment()

	})

	AfterEach(func() {
		err := httpServer.Shutdown(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("successful scenarios", func() {

		BeforeEach(func() {
			var err error
			httpServer, err = NewHappyServer()
			Expect(err).ToNot(HaveOccurred())

			createBackend(httpServer)
		})

		It("should get registration status ", func() {
			// when
			status, err := backend.GetRegistrationStatus(context.TODO(), deviceID, namespace)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(BeEquivalentTo("registered"))
		})

		It("should get target namespace ", func() {
			// when
			status, err := backend.GetTargetNamespace(context.TODO(), deviceID, initialNamespace, false)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(namespace))
		})

		It("should get configuration ", func() {
			// when
			configuration, err := backend.GetConfiguration(context.TODO(), deviceID, namespace)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(*configuration).To(Equal(deviceConfiguration))
		})

		It("should record heartbeat ", func() {
			// given
			hb := models.Heartbeat{
				Version: "1234",
			}

			// when
			status, err := backend.UpdateStatus(context.TODO(), deviceID, namespace, &hb)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(BeFalse())
			Expect(httpServer.recorder.GetHeartbeats()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      hb,
			}))
		})

		It("should process registration ", func() {
			// given
			reg := models.RegistrationInfo{
				Hardware: &models.HardwareInfo{Hostname: deviceID},
			}

			// when
			err := backend.Register(context.TODO(), deviceID, namespace, &reg)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(httpServer.recorder.GetRegistrations()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      reg,
			}))
		})

		It("should process enrolment for non-existing device", func() {
			// given
			ns := namespace
			enrolment := models.EnrolmentInfo{
				TargetNamespace: &ns,
			}

			// when
			exists, err := backend.Enrol(context.TODO(), deviceID, namespace, &enrolment)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(httpServer.recorder.GetEnrolments()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      enrolment,
			}))
		})

		It("should process enrolment for existing device", func() {
			// given
			ns := namespace
			enrolment := models.EnrolmentInfo{
				TargetNamespace: &ns,
			}

			// when
			exists, err := backend.Enrol(context.TODO(), existingDeviceID, namespace, &enrolment)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(httpServer.recorder.GetEnrolments()).To(ConsistOf(messageDescriptor{
				deviceID:  existingDeviceID,
				namespace: namespace,
				data:      enrolment,
			}))
		})
	})

	Describe("failing scenarios", func() {

		BeforeEach(func() {
			var err error
			httpServer, err = NewFailingServer()
			Expect(err).ToNot(HaveOccurred())
			createBackend(httpServer)
		})

		It("should fail getting registration status ", func() {
			// when
			_, err := backend.GetRegistrationStatus(context.TODO(), deviceID, namespace)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail getting target namespace ", func() {
			// when
			_, err := backend.GetTargetNamespace(context.TODO(), deviceID, namespace, false)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail getting configuration ", func() {
			// when
			_, err := backend.GetConfiguration(context.TODO(), deviceID, namespace)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail recording heartbeat ", func() {
			// given
			hb := models.Heartbeat{
				Version: "1234",
			}

			// when
			status, err := backend.UpdateStatus(context.TODO(), deviceID, namespace, &hb)

			// then
			Expect(err).To(HaveOccurred())
			Expect(status).To(BeTrue())
			Expect(httpServer.recorder.GetHeartbeats()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      hb,
			}))
		})

		It("should fail registration ", func() {
			// given
			reg := models.RegistrationInfo{
				Hardware: &models.HardwareInfo{Hostname: deviceID},
			}

			// when
			err := backend.Register(context.TODO(), deviceID, namespace, &reg)

			// then
			Expect(err).To(HaveOccurred())
			Expect(httpServer.recorder.GetRegistrations()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      reg,
			}))
		})

		It("should fail enrolment ", func() {
			// given
			ns := namespace
			enrolment := models.EnrolmentInfo{
				TargetNamespace: &ns,
			}

			// when
			exists, err := backend.Enrol(context.TODO(), deviceID, namespace, &enrolment)

			// then
			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(httpServer.recorder.GetEnrolments()).To(ConsistOf(messageDescriptor{
				deviceID:  deviceID,
				namespace: namespace,
				data:      enrolment,
			}))
		})
	})
})
