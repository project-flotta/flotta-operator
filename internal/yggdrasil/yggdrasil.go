package yggdrasil

import (
	"context"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/models"
	"github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var (
	defaultHeartbeatConfiguration = models.HeartbeatConfiguration{
		HardwareProfile: &models.HardwareProfileConfiguration{},
		PeriodSeconds:   60,
	}
)

type Handler struct {
	deviceRepository     *edgedevice.Repository
	deploymentRepository *edgedeployment.Repository
	initialNamespace     string
}

func NewYggdrasilHandler(deviceRepository *edgedevice.Repository, deploymentRepository *edgedeployment.Repository,
	initialNamespace string) *Handler {
	return &Handler{
		deviceRepository:     deviceRepository,
		deploymentRepository: deploymentRepository,
		initialNamespace:     initialNamespace,
	}
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return operations.NewGetDataMessageForDeviceNotFound()
		}
		return operations.NewGetDataMessageForDeviceInternalServerError()
	}

	edgeDeployments, err := h.deploymentRepository.ListForEdgeDevice(ctx, edgeDevice.Name, edgeDevice.Namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.FromContext(ctx).Error(err, "Cannot retrieve Edge Deployments")
			return operations.NewGetDataMessageForDeviceInternalServerError()
		}
	}
	dc := models.DeviceConfigurationMessage{
		DeviceID:      deviceID,
		Version:       edgeDevice.ResourceVersion,
		Configuration: &models.DeviceConfiguration{},
		Workloads:     h.toWorkloadList(ctx, edgeDeployments),
	}

	if edgeDevice.Spec.Heartbeat != nil {
		configuration := models.HeartbeatConfiguration{
			PeriodSeconds: edgeDevice.Spec.Heartbeat.PeriodSeconds,
		}
		if edgeDevice.Spec.Heartbeat.HardwareProfile != nil {
			configuration.HardwareProfile = &models.HardwareProfileConfiguration{
				Include: edgeDevice.Spec.Heartbeat.HardwareProfile.Include,
				Scope:   edgeDevice.Spec.Heartbeat.HardwareProfile.Scope,
			}
		} else {
			configuration.HardwareProfile = defaultHeartbeatConfiguration.HardwareProfile
		}
		dc.Configuration.Heartbeat = &configuration
	} else {
		dc.Configuration.Heartbeat = &defaultHeartbeatConfiguration
	}

	// TODO: Network optimization: Decide whether there is a need to return any payload based on difference between last applied configuration and current state in the cluster.
	message := models.Message{
		Type:      models.MessageTypeData,
		Directive: "device",
		MessageID: uuid.New().String(),
		Version:   1,
		Sent:      strfmt.DateTime(time.Now()),
		Content:   dc,
	}
	return operations.NewGetDataMessageForDeviceOK().WithPayload(&message)

}

func (h *Handler) PostControlMessageForDevice(ctx context.Context, params yggdrasil.PostControlMessageForDeviceParams) middleware.Responder {
	return operations.NewPostControlMessageForDeviceOK()
}

func (h *Handler) PostDataMessageForDevice(ctx context.Context, params yggdrasil.PostDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := log.FromContext(ctx).WithValues("DeviceID", deviceID)
	msg := params.Message
	switch msg.Directive {
	case "heartbeat":
		heartbeat := models.Heartbeat{}
		contentJson, _ := json.Marshal(msg.Content)
		json.Unmarshal(contentJson, &heartbeat)
		logger.Info("Received heartbeat", "content", heartbeat)
		edgeDevice, err := h.deviceRepository.Read(ctx, params.DeviceID, h.initialNamespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return operations.NewPostDataMessageForDeviceNotFound()
			}
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		edgeDevice.Status.LastSyncedResourceVersion = heartbeat.Version
		edgeDevice.Status.LastSeenTime = metav1.NewTime(time.Time(heartbeat.Time))
		edgeDevice.Status.Phase = heartbeat.Status
		edgeDevice, err = h.deviceRepository.UpdateStatus(ctx, *edgeDevice)
		if err != nil {
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		h.updateEdgeDeployments(ctx, heartbeat.Workloads)
	case "registration":
		_, err := h.deviceRepository.Read(ctx, params.DeviceID, h.initialNamespace)
		if err != nil {
			if errors.IsNotFound(err) {
				contentJson, _ := json.Marshal(msg.Content)
				registrationInfo := models.RegistrationInfo{}
				json.Unmarshal(contentJson, &registrationInfo)
				logger.Info("Received registration info", "content", registrationInfo)
				now := metav1.Now()
				device := v1alpha1.EdgeDevice{
					Spec: v1alpha1.EdgeDeviceSpec{
						RequestTime: &now,
					}}
				device.Name = params.DeviceID
				device.Namespace = h.initialNamespace
				device.Spec.OsImageId = registrationInfo.OsImageID
				edgeDevice, err := h.deviceRepository.Create(ctx, device)
				if err != nil {
					logger.Error(err, "Cannot save EdgeDevice")
					return operations.NewPostDataMessageForDeviceInternalServerError()
				}
				logger.Info("Created", "device", edgeDevice)
			}
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	default:
		logger.Info("Received unknown message", "message", msg)
	}
	return operations.NewPostDataMessageForDeviceOK()
}

func (h *Handler) toWorkloadList(ctx context.Context, deployments []v1alpha1.EdgeDeployment) models.WorkloadList {
	list := models.WorkloadList{}
	for _, deployment := range deployments {
		podSpec, err := yaml.Marshal(deployment.Spec.Pod.Spec)
		if err != nil {
			log.FromContext(ctx).Error(err, "Cannot marshal pod specification")
			continue
		}
		workload := models.Workload{
			Name:          deployment.Name,
			Specification: string(podSpec),
			Version:       deployment.ResourceVersion,
		}
		list = append(list, &workload)
	}
	return list
}

func (h *Handler) updateEdgeDeployments(ctx context.Context, workloadStatuses []*models.WorkloadStatus) {
	logger := log.FromContext(ctx)
	for _, status := range workloadStatuses {
		edgeDeployment, err := h.deploymentRepository.Read(ctx, status.Name, h.initialNamespace)
		if err != nil {
			logger.Error(err, "Cannot get Edge Deployment", "name", status.Name)
			continue
		}
		if edgeDeployment.Status.Phase != status.Status {
			edgeDeployment.Status.Phase = status.Status
			edgeDeployment.Status.LastTransitionTime = metav1.Now()
		}
		_, err = h.deploymentRepository.UpdateStatus(ctx, *edgeDeployment)
		if err != nil {
			logger.Error(err, "Cannot update Edge Deployment status")
			continue
		}
	}
}
