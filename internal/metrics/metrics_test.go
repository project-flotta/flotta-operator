package metrics_test

import (
	"github.com/project-flotta/flotta-operator/internal/metrics"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	dto "github.com/prometheus/client_model/go"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	numberOfEdgeDevicesSuccessfulRegisteredValue = 3
	numberOfEdgeDevicesFailedToRegisterValue     = 1
	numberOfEdgeDevicesUnregisteredValue         = 2
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

func findMetric(src []*dto.MetricFamily, query string) *dto.MetricFamily {
	for _, s := range src {
		if s.Name != nil && *s.Name == query {
			return s
		}
	}
	return nil
}

func validateMetric(query string, value int) {
	data, err := ctrlmetrics.Registry.Gather()
	Expect(err).NotTo(HaveOccurred())

	m := findMetric(data, query)
	Expect(m).NotTo(BeNil(), "metric for %s could not be found", query)
	Expect(m.Metric).To(HaveLen(1))
	Expect(m.Metric[0].Counter).NotTo(BeNil())
	Expect(m.Metric[0].Counter.Value).NotTo(BeNil())
	Expect(*m.Metric[0].Counter.Value).To(BeEquivalentTo(value))
}

var _ = Describe("Metrics", func() {
	var (
		m metrics.Metrics
	)

	BeforeEach(func() {
		m = metrics.New()
	})

	Context("EdgeDevice", func() {
		It("correctly passes calls to the IncEdgeDeviceFailedRegistration", func() {
			for i := 0; i < numberOfEdgeDevicesUnregisteredValue; i++ {
				m.IncEdgeDeviceUnregistration()
			}

			//then
			validateMetric(metrics.EdgeDeviceUnregistrationQuery, numberOfEdgeDevicesUnregisteredValue)
		})

		It("correctly passes calls to the IncEdgeDeviceSuccessfulRegistration", func() {
			//when
			for i := 0; i < numberOfEdgeDevicesSuccessfulRegisteredValue; i++ {
				m.IncEdgeDeviceSuccessfulRegistration()
			}

			//then
			validateMetric(metrics.EdgeDeviceSuccessfulRegistrationQuery, numberOfEdgeDevicesSuccessfulRegisteredValue)
		})

		It("correctly passes calls to the IncEdgeDeviceFailedRegistration", func() {
			for i := 0; i < numberOfEdgeDevicesFailedToRegisterValue; i++ {
				m.IncEdgeDeviceFailedRegistration()
			}

			//then
			validateMetric(metrics.EdgeDeviceFailedRegistrationQuery, numberOfEdgeDevicesFailedToRegisterValue)
		})
	})
})
