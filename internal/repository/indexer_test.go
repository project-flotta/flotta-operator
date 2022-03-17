package indexer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	flottalabels "github.com/project-flotta/flotta-operator/internal/labels"
	indexer "github.com/project-flotta/flotta-operator/internal/repository"
)

var _ = Describe("Index functions", func() {

	Context("Index func of edge deployment", func() {
		It("Creates keys by selector labels only", func() {
			// given
			deployment := managementv1alpha1.EdgeDeployment{}
			deployment.Labels = map[string]string{
				"foo":                                    "bar",
				flottalabels.SelectorLabelPrefix + "abc": "123",
				flottalabels.SelectorLabelPrefix + flottalabels.DeviceNameLabel: "xyz",
			}

			// when
			keys := indexer.DeploymentByDeviceIndexFunc(&deployment)

			// then
			Expect(keys).To(HaveLen(2))
			Expect(keys).Should(ConsistOf("abc", "xyz"))
		})
	})

	Context("Index func of edge device", func() {
		It("Creates keys by workload labels only", func() {
			// given
			edge := managementv1alpha1.EdgeDevice{}
			edge.Labels = map[string]string{
				"foo":                                    "bar",
				flottalabels.WorkloadLabelPrefix + "abc": "123",
			}

			// when
			keys := indexer.DeviceByWorkloadIndexFunc(&edge)

			// then
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).To(Equal("abc"))
		})
	})
})
