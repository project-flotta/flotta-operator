package metrics_test

import (
	"github.com/project-flotta/flotta-operator/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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

		It("has heartbeat metric registered", func() {
			// given
			name := "the-name-1"
			namespace := "the-namespace-1"

			// when
			m.RegisterDeviceCounter(namespace, name)

			// then
			err := ctrlmetrics.Registry.Register(prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: metrics.EdgeDeviceHeartbeatQuery,
					ConstLabels: prometheus.Labels{
						"deviceNamespace": namespace,
						"deviceID":        name,
					},
				}))
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(prometheus.AlreadyRegisteredError{}))
		})

		It("can have heartbeat metric registered multiple times", func() {
			// given
			name := "the-name-2"
			namespace := "the-namespace-2"

			// when
			m.RegisterDeviceCounter(namespace, name)
			m.RegisterDeviceCounter(namespace, name)
			m.RegisterDeviceCounter(namespace, name)

			// then
			err := ctrlmetrics.Registry.Register(prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: metrics.EdgeDeviceHeartbeatQuery,
					ConstLabels: prometheus.Labels{
						"deviceNamespace": namespace,
						"deviceID":        name,
					},
				}))
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(prometheus.AlreadyRegisteredError{}))
		})

		It("has heartbeat metric unregistered", func() {
			// given
			name := "the-name-3"
			namespace := "the-namespace-3"
			m.RegisterDeviceCounter(namespace, name)

			// when
			m.RemoveDeviceCounter(namespace, name)

			// then
			err := ctrlmetrics.Registry.Register(prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: metrics.EdgeDeviceHeartbeatQuery,
					ConstLabels: prometheus.Labels{
						"deviceNamespace": namespace,
						"deviceID":        name,
					},
				}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("records heartbeat presence", func() {
			// given
			name := "the-name-4"
			namespace := "the-namespace-4"

			// when
			m.RecordEdgeDevicePresence(namespace, name)

			// then
			data, err := ctrlmetrics.Registry.Gather()
			Expect(err).NotTo(HaveOccurred())

			mf := findMetric(data, metrics.EdgeDeviceHeartbeatQuery)
			Expect(m).NotTo(BeNil())

			for _, m := range mf.Metric {
				if hasLabel(m, "deviceID", name) && hasLabel(m, "deviceNamespace", namespace) {
					Expect(*m.Counter.Value).To(BeEquivalentTo(1))
					return
				}
			}

			Fail("Metric not found")
		})
	})
})

func hasLabel(metric *dto.Metric, key, value string) bool {
	for _, l := range metric.Label {
		if *l.Name == key && *l.Value == value {
			return true
		}
	}
	return false
}
