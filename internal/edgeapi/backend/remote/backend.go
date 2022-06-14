package remote

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/project-flotta/flotta-operator/backend/client"
	backendclient "github.com/project-flotta/flotta-operator/backend/client/backend"
	backendmodels "github.com/project-flotta/flotta-operator/backend/models"
	backendapi "github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/models"
)

type backend struct {
	timeout          time.Duration
	logger           *zap.SugaredLogger
	initialNamespace string
	backendApi       *client.FlottaBackendAPI
}

func NewBackend(initialNamespace string, backendApi *client.FlottaBackendAPI, timeout time.Duration, logger *zap.SugaredLogger) *backend {
	return &backend{
		timeout:          timeout,
		logger:           logger,
		initialNamespace: initialNamespace,
		backendApi:       backendApi,
	}
}

func (b *backend) GetRegistrationStatus(ctx context.Context, name, namespace string) (backendapi.RegistrationStatus, error) {
	payload, err := b.getRegistrationStatus(ctx, name, namespace)
	if err != nil {
		return "", err
	}
	return backendapi.RegistrationStatus(payload.Status), nil
}

func (b *backend) GetConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error) {
	request := backendclient.NewGetDeviceConfigurationParams().WithTimeout(b.timeout).
		WithDeviceID(name).
		WithNamespace(namespace)

	response, err := b.backendApi.Backend.GetDeviceConfiguration(ctx, request)
	if err != nil {
		b.logger.With("DeviceID", name, "Namespace", namespace).
			Warnf("Error while getting device configuration: %s", err.Error())
		return nil, err
	}
	return &response.Payload.DeviceConfiguration, nil
}

func (b *backend) Enrol(ctx context.Context, name, namespace string, enrolmentInfo *models.EnrolmentInfo) (bool, error) {
	request := backendclient.NewEnrolDeviceParams().WithTimeout(b.timeout).
		WithDeviceID(name).
		WithNamespace(namespace).
		WithEnrolmentInfo(*enrolmentInfo)

	existsResponse, _, err := b.backendApi.Backend.EnrolDevice(ctx, request)
	if err != nil {
		b.logger.With("DeviceID", name, "Namespace", namespace).
			Warnf("Error while enroling device: %s", err.Error())
		return false, err
	}

	if existsResponse != nil {
		return true, nil
	}
	return false, nil
}

func (b *backend) GetTargetNamespace(ctx context.Context, name, namespace string, _ bool) (string, error) {
	payload, err := b.getRegistrationStatus(ctx, name, namespace)
	if err != nil {
		return "", err

	}
	return payload.Namespace, nil
}

func (b *backend) Register(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error {
	request := backendclient.NewRegisterDeviceParams().WithTimeout(b.timeout).
		WithDeviceID(name).
		WithNamespace(namespace).
		WithRegistrationInfo(*registrationInfo)

	_, err := b.backendApi.Backend.RegisterDevice(ctx, request)
	if err != nil {
		b.logger.With("DeviceID", name, "Namespace", namespace).
			Warnf("Error while registering device: %s", err.Error())
		return err
	}
	return nil
}

func (b *backend) UpdateStatus(ctx context.Context, name, namespace string, heartbeat *models.Heartbeat) (bool, error) {
	request := backendclient.NewUpdateHeartBeatParams().WithTimeout(b.timeout).
		WithDeviceID(name).
		WithNamespace(namespace).
		WithHeartbeat(*heartbeat)

	_, err := b.backendApi.Backend.UpdateHeartBeat(ctx, request)
	if err != nil {
		b.logger.With("DeviceID", name, "Namespace", namespace).
			Warnf("Error while updating status: %s", err.Error())
		return true, err
	}
	return false, nil
}

func (b *backend) getRegistrationStatus(ctx context.Context, name string, namespace string) (*backendmodels.DeviceRegistrationStatusResponse, error) {
	request := backendclient.NewGetRegistrationStatusParams().WithTimeout(b.timeout).
		WithDeviceID(name).
		WithNamespace(namespace)

	response, err := b.backendApi.Backend.GetRegistrationStatus(ctx, request)
	if err != nil {
		b.logger.With("DeviceID", name, "Namespace", namespace).
			Warnf("Error while getting registration status: %s", err.Error())
		return nil, err
	}
	payload := response.Payload
	return payload, nil
}
func (b *backend) GetPlaybookExecutions(ctx context.Context, name string, namespace string) (*models.PlaybookExecutionsResponse, error) {
	request := backendclient.NewGetPlaybookExecutionsParams().
		WithNamespace(namespace).
		WithDeviceID(name)

	response, err := b.backendApi.Backend.GetPlaybookExecutions(ctx, request)
	if err != nil {
		b.logger.With("PlaybookExecutionID", name, "Namespace", namespace).
			Warnf("Error while getting playbook execution: %s", err.Error())
		return nil, err
	}
	return &response.Payload, nil
}
