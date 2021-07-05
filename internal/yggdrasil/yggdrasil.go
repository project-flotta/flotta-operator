package yggdrasil

import (
	"context"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Handler struct {
	deviceRepository *edgedevice.Repository
	initialNamespace string
}

func NewYggdrasilHandler(deviceRepository *edgedevice.Repository, initialNamespace string) *Handler {
	return &Handler{
		deviceRepository: deviceRepository,
		initialNamespace: initialNamespace,
	}
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	logger := log.FromContext(ctx).WithValues("DeviceID", params.DeviceID)
	logger.Info("Requested control message for device")
	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	logger := log.FromContext(ctx).WithValues("DeviceID", params.DeviceID)
	logger.Info("Requested data for device")
	h.deviceRepository.Read(ctx, params.DeviceID, h.initialNamespace)
	return operations.NewGetDataMessageForDeviceOK()
}

func (h *Handler) PostControlMessageForDevice(ctx context.Context, params yggdrasil.PostControlMessageForDeviceParams) middleware.Responder {
	logger := log.FromContext(ctx).WithValues("DeviceID", params.DeviceID)
	logger.Info("Received control message for device", "Message", params.Message)
	return operations.NewPostControlMessageForDeviceOK()
}

func (h *Handler) PostDataMessageForDevice(ctx context.Context, params yggdrasil.PostDataMessageForDeviceParams) middleware.Responder {
	panic("implement me")
}
