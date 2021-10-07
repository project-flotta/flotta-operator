package utils_test

import (
	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {

	Context("HasFinalizer", func() {
		const finalizer = "my-finalizer"

		It("should find finalizer in CR", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, finalizer)

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeTrue())
		})

		It("should find finalizer in CR when among other finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, "other", finalizer, "and-another")

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeTrue())
		})

		It("should not find finalizer in CR when no finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeFalse())
		})

		It("should not find finalizer in CR when not among other finalizers", func() {
			// given
			edgeDevice := managementv1alpha1.EdgeDevice{}
			edgeDevice.Finalizers = append(edgeDevice.Finalizers, "other", "and-another")

			// when
			found := utils.HasFinalizer(&edgeDevice.ObjectMeta, finalizer)

			// then
			Expect(found).To(BeFalse())
		})
	})
})
