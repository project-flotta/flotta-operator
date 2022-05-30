package backend

import (
	"context"

	"github.com/project-flotta/flotta-operator/models"
)

const (
	Registered   = RegistrationStatus("registered")
	Unregistered = RegistrationStatus("unregistered")
	Unknown      = RegistrationStatus("unknown")
)

type RegistrationStatus string

type Notification struct {
	DeviceID  string
	Namespace string
	Heartbeat *models.Heartbeat
	Retry     int32
}

// EdgeDeviceBackend represents API provided by data storage service to support edge device lifecycle.
type EdgeDeviceBackend interface {
	// GetRegistrationStatus responds with status of a device registration: {enrolled, registered, unregistered}
	GetRegistrationStatus(ctx context.Context, name, namespace string) (RegistrationStatus, error)

	// GetConfiguration provides complete Edge Device configuration that should be applied to the device
	GetConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error)

	// Enrol records device willingness to be connected to the cluster.
	Enrol(ctx context.Context, name string, enrolmentInfo *models.EnrolmentInfo) (bool, error)

	// GetTargetNamespace returns the namespace the device should belong to.
	GetTargetNamespace(ctx context.Context, name, namespace string, matchesCertificate bool) (string, error)

	// FinalizeRegistration is called during device registration request handling, after mTLS certificate has
	// been correctly issued.
	// The responsibility of the method is to potentially record information that the device is finally registered and
	// what hardware configuration it has.
	FinalizeRegistration(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error

	// UpdateStatus records current state of the device sent in a heartbeat message
	// (i.e. workload status, events reported by the device, OS upgrade status).
	UpdateStatus(ctx context.Context, notification Notification) (bool, error)
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
