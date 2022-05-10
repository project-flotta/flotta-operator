package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("EdgeDevice Webhook", func() {
	var (
		device EdgeDevice
	)
	BeforeEach(func() {
		device = EdgeDevice{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EdgeDeviceSpec{
				OsInformation: nil,
				RequestTime:   nil,
				Heartbeat:     nil,
				Storage:       &Storage{S3: &S3Storage{}},
				Metrics:       nil,
				LogCollection: nil,
			},
			Status: EdgeDeviceStatus{},
		}
	})

	Describe("EdgeDevice validating webhook", func() {
		Context("conflicting s3.secretName s3.configMapName s3.createOBC fields", func() {
			It("should fail to create", func() {
				// given
				device.Spec.Storage.S3.SecretName = "secret"
				device.Spec.Storage.S3.ConfigMapName = "cm"
				device.Spec.Storage.S3.CreateOBC = true
				// then
				Expect(device.ValidateCreate()).To(HaveOccurred())
			})
			It("should fail to update", func() {
				// given
				device.Spec.Storage.S3.SecretName = "secret"
				device.Spec.Storage.S3.ConfigMapName = "cm"
				device.Spec.Storage.S3.CreateOBC = true
				// then
				Expect(device.ValidateUpdate(nil)).To(HaveOccurred())
			})
		})
	})
})
