package yggdrasil

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	"github.com/jakub-dzon/k4e-operator/models"
	"github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const YggdrasilConnectionFinalizer = "yggdrasil-connection-finalizer"
const YggdrasilWorkloadFinalizer = "yggdrasil-workload-finalizer"

var (
	defaultHeartbeatConfiguration = models.HeartbeatConfiguration{
		HardwareProfile: &models.HardwareProfileConfiguration{},
		PeriodSeconds:   60,
	}
)

type Handler struct {
	deviceRepository     *edgedevice.Repository
	deploymentRepository *edgedeployment.Repository
	claimer              *storage.Claimer
	initialNamespace     string
}

func NewYggdrasilHandler(deviceRepository *edgedevice.Repository, deploymentRepository *edgedeployment.Repository, claimer *storage.Claimer, initialNamespace string) *Handler {
	return &Handler{
		deviceRepository:     deviceRepository,
		deploymentRepository: deploymentRepository,
		claimer:              claimer,
		initialNamespace:     initialNamespace,
	}
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return operations.NewGetControlMessageForDeviceNotFound()
		}
		return operations.NewGetControlMessageForDeviceInternalServerError()
	}
	// Send disconnect only if YggdrasilWorkloadFinalizer was already processed and removed
	if edgeDevice.DeletionTimestamp != nil && !utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
		message, err := h.createDisconnectCommand()
		if err != nil {
			return operations.NewGetControlMessageForDeviceInternalServerError()
		}
		err = h.deviceRepository.RemoveFinalizer(ctx, edgeDevice, YggdrasilConnectionFinalizer)
		if err != nil {
			return operations.NewGetControlMessageForDeviceInternalServerError()
		}
		return operations.NewGetControlMessageForDeviceOK().WithPayload(message)
	}
	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := log.FromContext(ctx).WithValues("DeviceID", deviceID)
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return operations.NewGetDataMessageForDeviceNotFound()
		}
		return operations.NewGetDataMessageForDeviceInternalServerError()
	}
	var workloadList models.WorkloadList

	if edgeDevice.DeletionTimestamp == nil {
		var edgeDeployments []v1alpha1.EdgeDeployment

		for _, deployment := range edgeDevice.Status.Deployments {
			edgeDeployment, err := h.deploymentRepository.Read(ctx, deployment.Name, edgeDevice.Namespace)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.FromContext(ctx).Error(err, "Cannot retrieve Edge Deployments")
					return operations.NewGetDataMessageForDeviceInternalServerError()
				}
				continue
			}
			edgeDeployments = append(edgeDeployments, *edgeDeployment)
		}

		workloadList = h.toWorkloadList(ctx, edgeDeployments)
	} else {
		if utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
			err := h.deviceRepository.RemoveFinalizer(ctx, edgeDevice, YggdrasilWorkloadFinalizer)
			if err != nil {
				return operations.NewGetDataMessageForDeviceInternalServerError()
			}
		}
	}

	dc := models.DeviceConfigurationMessage{
		DeviceID:      deviceID,
		Version:       edgeDevice.ResourceVersion,
		Configuration: &models.DeviceConfiguration{},
		Workloads:     workloadList,
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

	if edgeDevice.Status.DataOBC != nil && len(*edgeDevice.Status.DataOBC) > 0 {
		storageConf, err := h.claimer.GetStorageConfiguration(ctx, edgeDevice)
		if err != nil {
			logger.Error(err, "Failed to get storage configuration for device", "device", deviceID)
		} else {
			dc.Configuration.Storage = &models.StorageConfiguration{
				S3: storageConf,
			}
		}
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
		if heartbeat.Hardware != nil {
			edgeDevice.Status.Hardware = mapHardware(ctx, heartbeat.Hardware)
		}
		deployments := h.updateDeploymentStatuses(edgeDevice.Status.Deployments, heartbeat.Workloads)
		edgeDevice.Status.Deployments = deployments
		err = h.deviceRepository.UpdateStatus(ctx, edgeDevice)
		if err != nil {
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	case "registration":
		_, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
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
					},
				}
				device.Name = deviceID
				device.Namespace = h.initialNamespace
				device.Spec.OsImageId = registrationInfo.OsImageID
				device.Finalizers = []string{YggdrasilConnectionFinalizer, YggdrasilWorkloadFinalizer}
				err := h.deviceRepository.Create(ctx, &device)
				if err != nil {
					logger.Error(err, "Cannot save EdgeDevice")
					return operations.NewPostDataMessageForDeviceInternalServerError()
				}

				device.Status = v1alpha1.EdgeDeviceStatus{
					Hardware: mapHardware(ctx, registrationInfo.Hardware),
				}

				// TODO: when controller starts updating the EdgeDevice CR the status update below will need to be
				// executed in a retry loop to overcome potential optimistic locking problems.
				err = h.deviceRepository.UpdateStatus(ctx, &device)
				if err != nil {
					logger.Error(err, "Cannot update EdgeDevice status")
					return operations.NewPostDataMessageForDeviceInternalServerError()
				}
				logger.Info("Created", "device", device)
			}
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	default:
		logger.Info("Received unknown message", "message", msg)
	}
	return operations.NewPostDataMessageForDeviceOK()
}

func (h *Handler) updateDeploymentStatuses(oldDeployments []v1alpha1.Deployment, workloads []*models.WorkloadStatus) []v1alpha1.Deployment {
	deploymentMap := make(map[string]v1alpha1.Deployment)
	for _, deploymentStatus := range oldDeployments {
		deploymentMap[deploymentStatus.Name] = deploymentStatus
	}
	for _, status := range workloads {
		if deployment, ok := deploymentMap[status.Name]; ok {
			if string(deployment.Phase) != status.Status {
				deployment.Phase = v1alpha1.EdgeDeploymentPhase(status.Status)
				deployment.LastTransitionTime = metav1.Now()
			}
			deploymentMap[status.Name] = deployment
		}
	}
	var deployments []v1alpha1.Deployment
	for _, deployment := range deploymentMap {
		deployments = append(deployments, deployment)
	}
	return deployments
}

func mapHardware(ctx context.Context, hardware *models.HardwareInfo) *v1alpha1.Hardware {
	if hardware == nil {
		return nil
	}
	logger := log.FromContext(ctx)

	var disks []*v1alpha1.Disk
	err := utils.Copy(hardware.Disks, &disks)
	if err != nil {
		logger.Error(err, "Cannot map Disks")
	}
	var gpus []*v1alpha1.Gpu
	err = utils.Copy(hardware.Gpus, &gpus)
	if err != nil {
		logger.Error(err, "Cannot map Gpus")
	}

	var interfaces []*v1alpha1.Interface
	err = utils.Copy(hardware.Interfaces, &interfaces)
	if err != nil {
		logger.Error(err, "Cannot map Interfaces")
	}
	hw := v1alpha1.Hardware{
		Hostname: hardware.Hostname,

		Gpus:       gpus,
		Disks:      disks,
		Interfaces: interfaces,
	}
	if hardware.Boot != nil {
		hw.Boot = &v1alpha1.Boot{
			CurrentBootMode: hardware.Boot.CurrentBootMode,
			PxeInterface:    hardware.Boot.PxeInterface,
		}
	}

	cpu := hardware.CPU
	if cpu != nil {
		hw.CPU = &v1alpha1.CPU{
			Architecture: cpu.Architecture,
			Count:        cpu.Count,
			Flags:        cpu.Flags,
			Frequency:    fmt.Sprintf("%.2f", cpu.Frequency),
			ModelName:    cpu.ModelName,
		}
	}

	memory := hardware.Memory
	if memory != nil {
		hw.Memory = &v1alpha1.Memory{
			PhysicalBytes: memory.PhysicalBytes,
			UsableBytes:   memory.UsableBytes,
		}
	}

	systemVendor := hardware.SystemVendor
	if systemVendor != nil {
		hw.SystemVendor = &v1alpha1.SystemVendor{
			Manufacturer: systemVendor.Manufacturer,
			ProductName:  systemVendor.ProductName,
			SerialNumber: systemVendor.SerialNumber,
			Virtual:      systemVendor.Virtual,
		}
	}

	if err != nil {
		logger.Error(err, "Can't translate")
	}
	return &hw
}

func (h *Handler) toWorkloadList(ctx context.Context, deployments []v1alpha1.EdgeDeployment) models.WorkloadList {
	list := models.WorkloadList{}
	for _, deployment := range deployments {
		if deployment.DeletionTimestamp != nil {
			continue
		}
		podSpec, err := yaml.Marshal(deployment.Spec.Pod.Spec)
		if err != nil {
			log.FromContext(ctx).Error(err, "Cannot marshal pod specification")
			continue
		}
		workload := models.Workload{
			Name:          deployment.Name,
			Specification: string(podSpec),
		}
		list = append(list, &workload)
	}
	return list
}

func (h *Handler) createDisconnectCommand() (*models.Message, error) {
	command := struct {
		Command   string            `json:"command"`
		Arguments map[string]string `json:"arguments"`
	}{
		Command: "disconnect",
	}

	return &models.Message{
		Type:      models.MessageTypeCommand,
		MessageID: uuid.New().String(),
		Version:   1,
		Sent:      strfmt.DateTime(time.Now()),
		Content:   command,
	}, nil
}
