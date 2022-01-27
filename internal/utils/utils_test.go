package utils_test

import (
	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	Context("NormalizeLabel", func() {
		table.DescribeTable("should fail for an invalid format", func(tested string) {
			result, err := utils.NormalizeLabel(tested)
			Expect(result).To(Equal(""))
			Expect(err).To(HaveOccurred())
		},
			table.Entry("Empty string", ""),
			table.Entry("Non-alphanumeric characters", "$!@#$!@#$%"),
			table.Entry("Only dashes without alphanumeric characters", "-----"),
		)

		table.DescribeTable("should normalize given label to expected format", func(tested string, expected string) {
			result, err := utils.NormalizeLabel(tested)
			Expect(result).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		},
			table.Entry("CPU model", "Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz", "intelrcoretmi7-8665ucpu1.90ghz"),
			table.Entry("CPU arch", "x86_64", "x86_64"),
			table.Entry("Serial", "PF20YKWG;", "pf20ykwg"),
		)
	})
})
