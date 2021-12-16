package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// When adding metric names, see https://prometheus.io/docs/practices/naming/#metric-names
const (
	EdgeDeviceSuccessfulRegistrationQuery = "k4e_operator_edge_devices_successful_registration"
	EdgeDeviceFailedRegistrationQuery     = "k4e_operator_edge_devices_failed_registration"
	CreatedOBCQuery                       = "k4e_operator_created_obc"
)

var (
	registeredEdgeDevices = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceSuccessfulRegistrationQuery,
			Help: "Number of successful registration EdgeDevices",
		},
	)
	failedToCompleteRegistrationEdgeDevices = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceFailedRegistrationQuery,
			Help: "Number of failed registration EdgeDevices",
		},
	)
	createdOBC = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: CreatedOBCQuery,
			Help: "Number of created OBCs",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		registeredEdgeDevices,
		failedToCompleteRegistrationEdgeDevices,
		createdOBC,
	)
}

//go:generate mockgen -source=metrics.go -package=metrics -destination=mock_metrics_api.go

// Metrics is an interface representing a prometheus client for the Special Resource Operator
type Metrics interface {
	IncEdgeDeviceSuccessfulRegistration()
	IncEdgeDeviceFailedRegistration()
	IncCreatedOBCs()
}

func New() Metrics {
	return &metricsImpl{}
}

type metricsImpl struct{}

func (m *metricsImpl) IncEdgeDeviceSuccessfulRegistration() {
	registeredEdgeDevices.Inc()
}
func (m *metricsImpl) IncEdgeDeviceFailedRegistration() {
	failedToCompleteRegistrationEdgeDevices.Inc()
}
func (m *metricsImpl) IncCreatedOBCs() {
	createdOBC.Inc()
}
