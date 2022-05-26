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

type HeartbeatHandler interface {
	Process(ctx context.Context, notification Notification) (bool, error)
}

type Backend interface {
	ShouldEdgeDeviceBeUnregistered(ctx context.Context, name, namespace string) (bool, error)
	GetDeviceConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error)
	EnrolEdgeDevice(ctx context.Context, name string, enrolmentInfo *models.EnrolmentInfo) (bool, error)
	InitializeEdgeDeviceRegistration(ctx context.Context, name, namespace string, matchesCertificate bool) (bool, string, error)
	FinalizeEdgeDeviceRegistration(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error
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
