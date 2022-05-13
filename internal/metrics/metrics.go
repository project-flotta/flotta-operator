package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// When adding metric names, see https://prometheus.io/docs/practices/naming/#metric-names
const (
	EdgeDeviceSuccessfulRegistrationQuery = "flotta_operator_edge_devices_successful_registration"
	EdgeDeviceFailedRegistrationQuery     = "flotta_operator_edge_devices_failed_registration"
	EdgeDeviceUnregistrationQuery         = "flotta_operator_edge_devices_unregistration"
	EdgeDeviceHeartbeatQuery              = "flotta_operator_edge_devices_heartbeat"
	EdgeDeviceFailedAuthentication        = "flotta_operator_edge_devices_failed_authentication"
	EdgeDeviceInvalidOwner                = "flotta_operator_edge_devices_invalid_owner"
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
	unregisteredEdgeDevices = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceUnregistrationQuery,
			Help: "Number of unregistered EdgeDevices",
		},
	)

	failedAuthenticationCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceFailedAuthentication,
			Help: "Counts the number of devices that failed to authenticate",
		},
	)

	invalidOwnerCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceInvalidOwner,
			Help: "Counts the number of times a device ID mismatches the ID stored in the context",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		registeredEdgeDevices,
		failedToCompleteRegistrationEdgeDevices,
		unregisteredEdgeDevices,
		failedAuthenticationCounter,
		invalidOwnerCounter,
	)
}

//go:generate mockgen -source=metrics.go -package=metrics -destination=mock_metrics_api.go

// Metrics is an interface representing a prometheus client for the Special Resource Operator
type Metrics interface {
	IncEdgeDeviceSuccessfulRegistration()
	IncEdgeDeviceFailedRegistration()
	IncEdgeDeviceUnregistration()
	RecordEdgeDevicePresence(namespace, name string)
	RemoveDeviceCounter(namespace, name string)
	RegisterDeviceCounter(namespace string, name string)
	IncEdgeDeviceFailedAuthenticationCounter()
	IncEdgeDeviceInvalidOwnerCounter()
}

func New() Metrics {
	return &metricsImpl{
		devices: sync.Map{},
	}
}

type metricsImpl struct {
	devices sync.Map
}

func (m *metricsImpl) RecordEdgeDevicePresence(namespace, name string) {
	m.registerDeviceCounter(namespace, name).Inc()
}

func (m *metricsImpl) RegisterDeviceCounter(namespace string, name string) {
	m.registerDeviceCounter(namespace, name)
}

func (m *metricsImpl) IncEdgeDeviceSuccessfulRegistration() {
	registeredEdgeDevices.Inc()
}

func (m *metricsImpl) IncEdgeDeviceFailedRegistration() {
	failedToCompleteRegistrationEdgeDevices.Inc()
}
func (m *metricsImpl) IncEdgeDeviceUnregistration() {
	unregisteredEdgeDevices.Inc()
}

func (m *metricsImpl) IncEdgeDeviceFailedAuthenticationCounter() {
	failedAuthenticationCounter.Inc()
}

func (m *metricsImpl) IncEdgeDeviceInvalidOwnerCounter() {
	invalidOwnerCounter.Inc()
}

func (m *metricsImpl) RemoveDeviceCounter(namespace, name string) {
	if counter, ok := m.devices.LoadAndDelete(deviceKey(namespace, name)); ok {
		metrics.Registry.Unregister(counter.(prometheus.Counter)) //nolint
	}
}

func (m *metricsImpl) registerDeviceCounter(namespace, name string) prometheus.Counter {
	key := deviceKey(namespace, name)
	collector, loaded := m.devices.LoadOrStore(key, prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: EdgeDeviceHeartbeatQuery,
			ConstLabels: prometheus.Labels{
				"deviceNamespace": namespace,
				"deviceID":        name,
			},
		}))

	counter := collector.(prometheus.Counter) //nolint
	if !loaded {
		metrics.Registry.MustRegister(counter)
	}
	return counter
}

func deviceKey(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
