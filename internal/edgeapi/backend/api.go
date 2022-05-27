package backend

import (
	"context"

	"github.com/project-flotta/flotta-operator/models"
)

type Notification struct {
	DeviceID  string
	Namespace string
	Heartbeat *models.Heartbeat
	Retry     int32
}

//go:generate mockgen -package=backend -destination=mock_heartbeat-handler.go . HeartbeatHandler
type HeartbeatHandler interface {
	Process(ctx context.Context, notification Notification) (bool, error)
}

// Backend represents API provided by data storage service to support edge device lifecycle.
type Backend interface {
	// ShouldEdgeDeviceBeUnregistered responds with true, when the device identified with name and namespace should be
	// instructed to execute de-registration procedure
	ShouldEdgeDeviceBeUnregistered(ctx context.Context, name, namespace string) (bool, error)

	// GetDeviceConfiguration provides complete Edge Device configuration that should be applied to the device
	GetDeviceConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error)

	// EnrolEdgeDevice records device willingness to be connected to the cluster.
	EnrolEdgeDevice(ctx context.Context, name string, enrolmentInfo *models.EnrolmentInfo) (bool, error)

	// InitializeEdgeDeviceRegistration is called when device sends registration request (either to issue the mTLS certificate
	// for the first time or renew it) and has to return information whether it is handling first registration request from the device,
	// and namespace the device should be created in.
	InitializeEdgeDeviceRegistration(ctx context.Context, name, namespace string, matchesCertificate bool) (bool, string, error)

	// FinalizeEdgeDeviceRegistration is called during device registration request handling, after mTLS certificate has
	// been correctly issued.
	// The responsibility of the method is to potentially record information that the device is finally registered and
	// what hardware configuration it has.
	FinalizeEdgeDeviceRegistration(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error

	// GetHeartbeatHandler provides implementation of a HeartbeatHandler that should record current state of the device sent in
	// (i.e. workload status, events reported by the device, OS upgrade status).
	GetHeartbeatHandler() HeartbeatHandler
}

type NotApproved struct {
	cause error
}

func NewNotApproved(err error) *NotApproved {
	return &NotApproved{
		cause: err,
	}
}

func (e *NotApproved) Error() string {
	return "not approved"
}

func (e *NotApproved) Unwrap() error {
	return e.cause
}
