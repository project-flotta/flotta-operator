package yggdrasil

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/hardware"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	"github.com/jakub-dzon/k4e-operator/models"
	"github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	deviceRepository     edgedevice.Repository
	deploymentRepository edgedeployment.Repository
	claimer              *storage.Claimer
	initialNamespace     string
}

func NewYggdrasilHandler(deviceRepository edgedevice.Repository, deploymentRepository edgedeployment.Repository, claimer *storage.Claimer, initialNamespace string) *Handler {
	return &Handler{
		deviceRepository:     deviceRepository,
		deploymentRepository: deploymentRepository,
		claimer:              claimer,
		initialNamespace:     initialNamespace,
	}
}

func (h *Handler) GetControlMessageForDevice(ctx context.Context, params yggdrasil.GetControlMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetControlMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device")
		return operations.NewGetControlMessageForDeviceInternalServerError()
	}

	if edgeDevice.DeletionTimestamp != nil && !utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
		err = h.deviceRepository.RemoveFinalizer(ctx, edgeDevice, YggdrasilConnectionFinalizer)
		if err != nil {
			return operations.NewGetControlMessageForDeviceInternalServerError()
		}
		message := h.createDisconnectCommand()
		return operations.NewGetControlMessageForDeviceOK().WithPayload(message)
	}
	return operations.NewGetControlMessageForDeviceOK()
}

func (h *Handler) GetDataMessageForDevice(ctx context.Context, params yggdrasil.GetDataMessageForDeviceParams) middleware.Responder {
	deviceID := params.DeviceID
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	edgeDevice, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("edge device is not found")
			return operations.NewGetDataMessageForDeviceNotFound()
		}
		logger.Error(err, "failed to get edge device")
		return operations.NewGetDataMessageForDeviceInternalServerError()
	}
	var workloadList models.WorkloadList

	if edgeDevice.DeletionTimestamp == nil {
		var edgeDeployments []v1alpha1.EdgeDeployment

		for _, deployment := range edgeDevice.Status.Deployments {
			edgeDeployment, err := h.deploymentRepository.Read(ctx, deployment.Name, edgeDevice.Namespace)
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "cannot retrieve Edge Deployments")
					return operations.NewGetDataMessageForDeviceInternalServerError()
				}
				continue
			}
			edgeDeployments = append(edgeDeployments, *edgeDeployment)
		}

		workloadList = h.toWorkloadList(logger, edgeDeployments)
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
			logger.Error(err, "failed to get storage configuration for device")
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
	logger := log.FromContext(ctx, "DeviceID", deviceID)
	msg := params.Message
	switch msg.Directive {
	case "heartbeat":
		heartbeat := models.Heartbeat{}
		contentJson, _ := json.Marshal(msg.Content)
		err := json.Unmarshal(contentJson, &heartbeat)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		logger.Info("received heartbeat", "content", heartbeat)
		edgeDevice, err := h.deviceRepository.Read(ctx, params.DeviceID, h.initialNamespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return operations.NewPostDataMessageForDeviceNotFound()
			}
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		err = h.updateDeviceStatus(ctx, edgeDevice, func(device *v1alpha1.EdgeDevice) {
			device.Status.LastSyncedResourceVersion = heartbeat.Version
			device.Status.LastSeenTime = metav1.NewTime(time.Time(heartbeat.Time))
			device.Status.Phase = heartbeat.Status
			if heartbeat.Hardware != nil {
				device.Status.Hardware = hardware.MapHardware(heartbeat.Hardware)
			}
			deployments := h.updateDeploymentStatuses(device.Status.Deployments, heartbeat.Workloads)
			device.Status.Deployments = deployments
		})
		if err != nil {
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
	case "registration":
		_, err := h.deviceRepository.Read(ctx, deviceID, h.initialNamespace)
		if err == nil {
			return operations.NewPostDataMessageForDeviceOK()
		}

		if !errors.IsNotFound(err) {
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		// register new edge device
		contentJson, _ := json.Marshal(msg.Content)
		registrationInfo := models.RegistrationInfo{}
		err = json.Unmarshal(contentJson, &registrationInfo)
		if err != nil {
			return operations.NewPostDataMessageForDeviceBadRequest()
		}
		logger.Info("received registration info", "content", registrationInfo)
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
		err = h.deviceRepository.Create(ctx, &device)
		if err != nil {
			logger.Error(err, "cannot save EdgeDevice")
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		err = h.updateDeviceStatus(ctx, &device, func(device *v1alpha1.EdgeDevice) {
			device.Status = v1alpha1.EdgeDeviceStatus{
				Hardware: hardware.MapHardware(registrationInfo.Hardware),
			}
		})

		if err != nil {
			logger.Error(err, "cannot update EdgeDevice status")
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		logger.Info("EdgeDevice created")
		return operations.NewPostDataMessageForDeviceOK()
	default:
		logger.Info("received unknown message", "message", msg)
		return operations.NewPostDataMessageForDeviceBadRequest()
	}
	return operations.NewPostDataMessageForDeviceOK()
}

func (h *Handler) updateDeviceStatus(ctx context.Context, device *v1alpha1.EdgeDevice, updateFunc func(d *v1alpha1.EdgeDevice)) error {
	patch := client.MergeFrom(device.DeepCopy())
	updateFunc(device)
	err := h.deviceRepository.PatchStatus(ctx, device, &patch)
	if err == nil {
		return nil
	}

	// retry patching the edge device status
	for i := 1; i < 4; i++ {
		time.Sleep(time.Duration(i*50) * time.Millisecond)
		device2, err := h.deviceRepository.Read(ctx, device.Name, device.Namespace)
		if err != nil {
			continue
		}
		patch = client.MergeFrom(device2.DeepCopy())
		updateFunc(device2)
		err = h.deviceRepository.PatchStatus(ctx, device2, &patch)
		if err == nil {
			return nil
		}
	}
	return err
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
			deployment.LastDataUpload = metav1.NewTime(time.Time(status.LastDataUpload))
			deploymentMap[status.Name] = deployment
		}
	}
	var deployments []v1alpha1.Deployment
	for _, deployment := range deploymentMap {
		deployments = append(deployments, deployment)
	}
	return deployments
}

func (h *Handler) toWorkloadList(logger logr.Logger, deployments []v1alpha1.EdgeDeployment) models.WorkloadList {
	list := models.WorkloadList{}
	for _, deployment := range deployments {
		if deployment.DeletionTimestamp != nil {
			continue
		}
		podSpec, err := yaml.Marshal(deployment.Spec.Pod.Spec)
		if err != nil {
			logger.Error(err, "cannot marshal pod specification", "deployment name", deployment.Name)
			continue
		}
		var data *models.DataConfiguration
		if deployment.Spec.Data != nil && len(deployment.Spec.Data.Paths) > 0 {
			var paths []*models.DataPath
			for _, path := range deployment.Spec.Data.Paths {
				paths = append(paths, &models.DataPath{Source: path.Source, Target: path.Target})
			}
			data = &models.DataConfiguration{Paths: paths}
		}
		workload := models.Workload{
			Name:          deployment.Name,
			Specification: string(podSpec),
			Data:          data,
		}
		list = append(list, &workload)
	}
	return list
}

func (h *Handler) createDisconnectCommand() *models.Message {
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
	}
}
