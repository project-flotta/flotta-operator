package devices

import (
	"context"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/models"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/devices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceHandler struct {
	edgeDeviceRepository *edgedevice.Repository
	initialNamespace     string
}

func NewDeviceHandler(edgeDeviceRepository *edgedevice.Repository, initialNamespace string) *DeviceHandler {
	return &DeviceHandler{edgeDeviceRepository: edgeDeviceRepository, initialNamespace: initialNamespace}
}

func (h *DeviceHandler) RegisterDevice(ctx context.Context, params operations.RegisterDeviceParams) middleware.Responder {
	now := metav1.Now()
	edgeDevice := v1alpha1.EdgeDevice{
		Spec: v1alpha1.EdgeDeviceSpec{
			OsImageId:   params.RegistrationInfo.OsImageID,
			RequestTime: &now,
		},
	}
	edgeDevice.Namespace = h.initialNamespace
	created, err := h.edgeDeviceRepository.Create(ctx, edgeDevice)
	if err != nil {
		return operations.NewRegisterDeviceInternalServerError()
	}
	return operations.NewRegisterDeviceOK().WithPayload(&models.RegistrationConfirmation{
		ID: created.Name,
	})
}

func (h *DeviceHandler) GetDeviceConfiguration(ctx context.Context, params operations.GetDeviceConfigurationParams) middleware.Responder {
	// TODO: retrieve CRs from the cluster and respond
	return operations.NewGetDeviceConfigurationOK().WithPayload(&models.DeviceConfiguration{DeviceID: params.DeviceID})
}
