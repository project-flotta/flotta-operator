package yggdrasil

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/url"
	"strings"

	"time"

	"github.com/jakub-dzon/k4e-operator/internal/k8sclient"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/hardware"
	"github.com/jakub-dzon/k4e-operator/internal/images"
	"github.com/jakub-dzon/k4e-operator/internal/metrics"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	"github.com/jakub-dzon/k4e-operator/models"
	"github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	operations "github.com/jakub-dzon/k4e-operator/restapi/operations/yggdrasil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	YggdrasilConnectionFinalizer = "yggdrasil-connection-finalizer"
	YggdrasilWorkloadFinalizer   = "yggdrasil-workload-finalizer"
	YggdrasilRegisterAuth        = 1
	YggdrasilCompleteAuth        = 0
)

var (
	defaultHeartbeatConfiguration = models.HeartbeatConfiguration{
		HardwareProfile: &models.HardwareProfileConfiguration{},
		PeriodSeconds:   60,
	}
)

type Handler struct {
	deviceRepository       edgedevice.Repository
	deploymentRepository   edgedeployment.Repository
	claimer                *storage.Claimer
	client                 k8sclient.K8sClient
	initialNamespace       string
	recorder               record.EventRecorder
	registryAuthRepository images.RegistryAuthAPI
	metrics                metrics.Metrics
}

type keyMapType = map[string]interface{}
type secretMapType = map[string]keyMapType

func NewYggdrasilHandler(deviceRepository edgedevice.Repository, deploymentRepository edgedeployment.Repository,
	claimer *storage.Claimer, k8sClient k8sclient.K8sClient, initialNamespace string, recorder record.EventRecorder, registryAuth images.RegistryAuthAPI, metrics metrics.Metrics) *Handler {
	return &Handler{
		deviceRepository:       deviceRepository,
		deploymentRepository:   deploymentRepository,
		claimer:                claimer,
		client:                 k8sClient,
		initialNamespace:       initialNamespace,
		recorder:               recorder,
		registryAuthRepository: registryAuth,
		metrics:                metrics,
	}
}

func isRegistrationURL(url *url.URL) bool {
	parts := strings.Split(url.Path, "/")
	if len(parts) == 0 {
		return false
	}

	last := parts[len(parts)-1]
	return last == "registration"
}

func (h *Handler) GetAuthType(r *http.Request) int {
	res := YggdrasilCompleteAuth
	if isRegistrationURL(r.URL) {
		res = YggdrasilRegisterAuth
	}
	return res
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
	var secretList models.SecretList

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
			if edgeDeployment.DeletionTimestamp == nil {
				edgeDeployments = append(edgeDeployments, *edgeDeployment)
			}
		}

		workloadList, err = h.toWorkloadList(ctx, logger, edgeDeployments, edgeDevice)
		if err != nil {
			return operations.NewGetDataMessageForDeviceInternalServerError()
		}
		secretList, err = h.createSecretList(ctx, logger, edgeDeployments, edgeDevice)
		if err != nil {
			logger.Error(err, "failed reading secrets for device deployments")
			return operations.NewGetDataMessageForDeviceInternalServerError()
		}
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
		Secrets:       secretList,
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

	err = h.setStorageConfiguration(ctx, edgeDevice, &dc)
	if err != nil {
		logger.Error(err, "failed to get storage configuration for device")
	}

	dc.Configuration.Metrics = getDeviceMetricsConfiguration(edgeDevice)

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

func getDeviceMetricsConfiguration(edgeDevice *v1alpha1.EdgeDevice) *models.MetricsConfiguration {
	var metricsConfig *models.MetricsConfiguration
	if edgeDevice.Spec.Metrics != nil {
		retention := edgeDevice.Spec.Metrics.Retention
		if retention != nil {
			metricsConfig = &models.MetricsConfiguration{
				Retention: &models.MetricsRetention{
					MaxHours: retention.MaxHours,
					MaxMib:   retention.MaxMiB,
				},
			}
		}
	}
	return metricsConfig
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
		logger.V(1).Info("received heartbeat", "content", heartbeat)
		edgeDevice, err := h.deviceRepository.Read(ctx, params.DeviceID, h.initialNamespace)
		if err != nil {
			if errors.IsNotFound(err) {
				return operations.NewPostDataMessageForDeviceNotFound()
			}
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}

		// Produce k8s events based on the device-worker events:
		events := heartbeat.Events
		for _, event := range events {
			if event == nil {
				continue
			}

			if event.Type == models.EventInfoTypeWarn {
				h.recorder.Event(edgeDevice, corev1.EventTypeWarning, event.Reason, event.Message)
			} else {
				h.recorder.Event(edgeDevice, corev1.EventTypeNormal, event.Reason, event.Message)
			}
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
		logger.V(1).Info("received registration info", "content", registrationInfo)
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
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		err = h.updateDeviceStatus(ctx, &device, func(device *v1alpha1.EdgeDevice) {
			device.Status = v1alpha1.EdgeDeviceStatus{
				Hardware: hardware.MapHardware(registrationInfo.Hardware),
			}
		})

		if err != nil {
			logger.Error(err, "cannot update EdgeDevice status")
			h.metrics.IncEdgeDeviceFailedRegistration()
			return operations.NewPostDataMessageForDeviceInternalServerError()
		}
		logger.Info("EdgeDevice created")
		h.metrics.IncEdgeDeviceSuccessfulRegistration()
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

func (h *Handler) toWorkloadList(ctx context.Context, logger logr.Logger, deployments []v1alpha1.EdgeDeployment, device *v1alpha1.EdgeDevice) (models.WorkloadList, error) {
	list := models.WorkloadList{}
	for _, deployment := range deployments {
		if deployment.DeletionTimestamp != nil {
			continue
		}
		spec := deployment.Spec
		podSpec, err := yaml.Marshal(spec.Pod.Spec)
		if err != nil {
			logger.Error(err, "cannot marshal pod specification", "deployment name", deployment.Name)
			continue
		}
		var data *models.DataConfiguration
		if spec.Data != nil && len(spec.Data.Paths) > 0 {
			var paths []*models.DataPath
			for _, path := range spec.Data.Paths {
				paths = append(paths, &models.DataPath{Source: path.Source, Target: path.Target})
			}
			data = &models.DataConfiguration{Paths: paths}
		}

		workload := models.Workload{
			Name:          deployment.Name,
			Specification: string(podSpec),
			Data:          data,
		}
		authFile, err := h.getAuthFile(ctx, spec.ImageRegistries, deployment.Namespace)
		if err != nil {
			msg := fmt.Sprintf("Auth file secret %s used by deployment %s/%s is missing", spec.ImageRegistries.AuthFileSecret.Name, deployment.Namespace, deployment.Name)
			h.recorder.Event(device, corev1.EventTypeWarning, "Misconfiguration", msg)
			logger.Error(err, msg)
			return nil, err
		}
		if authFile != "" {
			workload.ImageRegistries = &models.ImageRegistries{
				AuthFile: authFile,
			}
		}

		if spec.Metrics != nil && spec.Metrics.Port > 0 {
			workload.Metrics = &models.Metrics{
				Path:     spec.Metrics.Path,
				Port:     spec.Metrics.Port,
				Interval: spec.Metrics.Interval,
			}
			addedContainers := false
			containers := map[string]models.ContainerMetrics{}
			for container, metricConf := range spec.Metrics.Containers {
				containers[container] = models.ContainerMetrics{
					Disabled: metricConf.Disabled,
					Port:     metricConf.Port,
					Path:     metricConf.Path,
				}
				addedContainers = true
			}
			if addedContainers {
				workload.Metrics.Containers = containers
			}
		}
		list = append(list, &workload)
	}
	return list, nil
}

func (h *Handler) getAuthFile(ctx context.Context, imageRegistries *v1alpha1.ImageRegistriesConfiguration, defaultNamespace string) (string, error) {
	if imageRegistries != nil {
		if secretRef := imageRegistries.AuthFileSecret; secretRef != nil {
			namespace := secretRef.Namespace
			if secretRef.Namespace == "" {
				namespace = defaultNamespace
			}

			authFile, err := h.registryAuthRepository.GetAuthFileFromSecret(ctx, namespace, secretRef.Name)
			if err != nil {
				return "", err
			}
			return authFile, nil
		}
	}
	return "", nil
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

func (h *Handler) setStorageConfiguration(ctx context.Context,
	edgeDevice *v1alpha1.EdgeDevice, dc *models.DeviceConfigurationMessage) error {

	var storageConf *models.S3StorageConfiguration
	var err error

	if edgeDevice.Status.DataOBC != nil && len(*edgeDevice.Status.DataOBC) > 0 {
		storageConf, err = h.claimer.GetStorageConfiguration(ctx, edgeDevice)
	} else if storage.ShouldUseExternalConfig(edgeDevice) {
		storageConf, err = h.claimer.GetExternalStorageConfig(ctx, edgeDevice)
	}

	if err == nil && storageConf != nil {
		dc.Configuration.Storage = &models.StorageConfiguration{
			S3: storageConf,
		}
	}

	return err
}

func (h *Handler) readAndValidateSecret(ctx context.Context, secretName, secretNamespace string, secretKeys keyMapType) (*corev1.Secret, error) {
	optional := secretKeys == nil
	secretObj := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace}}
	err := h.client.Get(ctx, client.ObjectKeyFromObject(secretObj), secretObj)
	if err != nil {
		if errors.IsNotFound(err) && optional {
			return nil, nil
		}
		return nil, err
	}
	for key := range secretKeys {
		if secretObj.Data != nil {
			if _, ok := secretObj.Data[key]; ok {
				continue
			}
		}
		return nil, fmt.Errorf("missing secret key. secret: %s. key: %s. Namespace: %s", secretName, key, secretNamespace)
	}
	return secretObj, nil
}

func addSecretToSecretList(secretList *models.SecretList, secret *corev1.Secret) error {
	dataMap := map[string]string{}
	for name, value := range secret.Data {
		dataMap[name] = strfmt.Base64(value).String()
	}
	dataJson, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}
	*secretList = append(*secretList, &models.Secret{
		Data: string(dataJson),
		Name: secret.Name,
	})

	return nil
}

func (h *Handler) createSecretList(ctx context.Context, logger logr.Logger, deployments []v1alpha1.EdgeDeployment, device *v1alpha1.EdgeDevice) (models.SecretList, error) {
	list := models.SecretList{}

	// create map of secret names and keys
	secretMap := secretMapType{}
	for _, deployment := range deployments {
		podSpec := deployment.Spec.Pod.Spec
		allContainers := append(podSpec.InitContainers, podSpec.Containers...)
		for _, container := range allContainers {
			extractSecretsFromContainer(&container, secretMap)
		}
	}

	// read secrets and add to secrets list
	for name, keys := range secretMap {
		secretObj, err := h.readAndValidateSecret(ctx, name, device.Namespace, keys)
		if err != nil {
			return nil, err
		}
		if secretObj == nil {
			continue
		}
		err = addSecretToSecretList(&list, secretObj)
		if err != nil {
			return nil, err
		}
	}

	return list, nil
}

// return value maps a secret name to its mandatory keys. nil keys indicates optional secret
func extractSecretsFromContainer(container *corev1.Container, secretMap secretMapType) {
	extractSecretsFromEnvFrom(container.EnvFrom, secretMap)
	extractSecretsFromEnv(container.Env, secretMap)
}

func extractSecretsFromEnvFrom(envFrom []corev1.EnvFromSource, secretMap secretMapType) {
	for _, envFrom := range envFrom {
		if envFrom.SecretRef == nil {
			continue
		}
		var optional bool
		if envFrom.SecretRef.Optional != nil {
			optional = *envFrom.SecretRef.Optional
		}
		if keys, ok := secretMap[envFrom.SecretRef.Name]; ok {
			if !optional && keys == nil {
				secretMap[envFrom.SecretRef.Name] = keyMapType{}
			}
		} else {
			if optional {
				secretMap[envFrom.SecretRef.Name] = nil
			} else {
				secretMap[envFrom.SecretRef.Name] = keyMapType{}
			}
		}
	}
}

func extractSecretsFromEnv(env []corev1.EnvVar, secretMap secretMapType) {
	for _, envVar := range env {
		if envVar.ValueFrom == nil || envVar.ValueFrom.SecretKeyRef == nil {
			continue
		}
		keyRef := envVar.ValueFrom.SecretKeyRef
		var optional bool
		if keyRef.Optional != nil {
			optional = *keyRef.Optional
		}
		if keys, ok := secretMap[keyRef.Name]; ok {
			if !optional {
				if keys == nil {
					secretMap[keyRef.Name] = keyMapType{keyRef.Key: nil}
				} else {
					keys[keyRef.Key] = nil
				}
			}
		} else {
			if optional {
				secretMap[keyRef.Name] = nil
			} else {
				secretMap[keyRef.Name] = keyMapType{keyRef.Key: nil}
			}
		}
	}
}
