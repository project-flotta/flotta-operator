package backend

import (
	"context"
	"github.com/project-flotta/flotta-operator/models"
)

type Backend interface {
	ShouldEdgeDeviceBeUnregistered(ctx context.Context, name, namespace string) (bool, error)
	GetDeviceConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error)
	EnrolEdgeDevice(ctx context.Context, name string, enrolmentInfo *models.EnrolmentInfo) (bool, error)
	InitializeEdgeDeviceRegistration(ctx context.Context, name, namespace string, matchesCertificate bool) (bool, string, error)
	FinalizeEdgeDeviceRegistration(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error
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
