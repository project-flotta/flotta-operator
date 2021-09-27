package utils_test

import (
	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Context("Copy", func() {

		type fromType struct {
			A string
			B bool
			C int
		}

		type toType struct {
			A string
			B bool
			C int
		}

		It("should copy nil", func() {
			// given
			var to []*toType

			// when
			err := utils.Copy(nil, &to)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(to).To(BeNil())
		})

		It("should copy object", func() {
			// given
			from := fromType{A: "a", B: true, C: 123}
			expected := toType{A: "a", B: true, C: 123}

			var to toType

			// when
			err := utils.Copy(from, &to)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(to).To(BeEquivalentTo(expected))
		})

		It("should fail copying object to slice", func() {
			// given
			from := fromType{A: "a", B: true, C: 123}

			var to []toType

			// when
			err := utils.Copy(from, &to)

			// then
			Expect(err).To(HaveOccurred())
		})

		It("should fail copying object with channel", func() {
			// given
			type fromType struct {
				A string
				B bool
				C int
				X chan<- struct{}
			}
			from := fromType{A: "a", B: true, C: 123}

			var to []toType

			// when
			err := utils.Copy(from, &to)

			// then
			Expect(err).To(HaveOccurred())
		})

		table.DescribeTable("should copy slice of objects", func(from []*fromType, expected []*toType) {
			// given
			var to []*toType

			// when
			err := utils.Copy(from, &to)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(to).To(BeEquivalentTo(expected))
		},
			table.Entry("no objects", []*fromType{}, []*toType{}),
			table.Entry("one object", []*fromType{
				{A: "a", B: true, C: 123},
			}, []*toType{
				{A: "a", B: true, C: 123},
			}),
			table.Entry("more objects", []*fromType{
				{A: "a", B: true, C: 123},
				{A: "aa", B: false, C: 321},
			}, []*toType{
				{A: "a", B: true, C: 123},
				{A: "aa", B: false, C: 321},
			}),
		)
	})

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
