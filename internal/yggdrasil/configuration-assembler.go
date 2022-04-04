package yggdrasil

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/project-flotta/flotta-operator/internal/k8sclient"
	"github.com/project-flotta/flotta-operator/internal/storage"

	"github.com/go-openapi/strfmt"
	"github.com/project-flotta/flotta-operator/internal/configmaps"
	"github.com/project-flotta/flotta-operator/internal/images"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/labels"
	"github.com/project-flotta/flotta-operator/models"
	corev1 "k8s.io/api/core/v1"
)

type configurationAssembler struct {
	allowLists             devicemetrics.AllowListGenerator
	claimer                *storage.Claimer
	client                 k8sclient.K8sClient
	configMaps             configmaps.ConfigMap
	deploymentRepository   edgedeployment.Repository
	recorder               record.EventRecorder
	registryAuthRepository images.RegistryAuthAPI
}

func (a *configurationAssembler) getDeviceConfiguration(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, logger logr.Logger) (*models.DeviceConfigurationMessage, error) {
	var workloadList models.WorkloadList
	var secretList models.SecretList
	if edgeDevice.DeletionTimestamp == nil {
		var edgeDeployments []v1alpha1.EdgeDeployment

		for _, deployment := range edgeDevice.Status.Deployments {
			edgeDeployment, err := a.deploymentRepository.Read(ctx, deployment.Name, edgeDevice.Namespace)
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "cannot retrieve Edge Deployments")
					return nil, err
				}
				continue
			}
			if edgeDeployment.DeletionTimestamp == nil {
				edgeDeployments = append(edgeDeployments, *edgeDeployment)
			}
		}
		var err error
		workloadList, err = a.toWorkloadList(ctx, logger, edgeDeployments, edgeDevice)
		if err != nil {
			return nil, err
		}
		secretList, err = a.createSecretList(ctx, logger, edgeDeployments, edgeDevice)
		if err != nil {
			logger.Error(err, "failed reading secrets for device deployments")
			return nil, err
		}
	}

	dc := models.DeviceConfigurationMessage{
		DeviceID:      edgeDevice.Name,
		Version:       edgeDevice.ResourceVersion,
		Configuration: &models.DeviceConfiguration{},
		Workloads:     workloadList,
		Secrets:       secretList,
	}

	dc.Configuration.Heartbeat = getHeartbeatConfiguration(edgeDevice)

	if edgeDevice.Spec.OsInformation != nil {
		dc.Configuration.Os = (*models.OsInformation)(edgeDevice.Spec.OsInformation)
	}

	err := a.setStorageConfiguration(ctx, edgeDevice, &dc)
	if err != nil {
		logger.Error(err, "failed to get storage configuration for device")
	}

	dc.Configuration.Metrics, err = a.getDeviceMetricsConfiguration(ctx, edgeDevice)
	if err != nil {
		logger.Error(err, "failed getting device metrics configuration")
		return nil, err
	}

	dc.Configuration.LogCollection, err = a.getDeviceLogConfig(ctx, edgeDevice)
	if err != nil {
		logger.Error(err, "failed getting device log configuration")
		return nil, err
	}

	return &dc, nil
}

func getHeartbeatConfiguration(edgeDevice *v1alpha1.EdgeDevice) *models.HeartbeatConfiguration {
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
		return &configuration
	}
	return &defaultHeartbeatConfiguration

}

func (a *configurationAssembler) setStorageConfiguration(ctx context.Context,
	edgeDevice *v1alpha1.EdgeDevice, dc *models.DeviceConfigurationMessage) error {

	var storageConf *models.S3StorageConfiguration
	var err error

	if edgeDevice.Status.DataOBC != nil && len(*edgeDevice.Status.DataOBC) > 0 {
		storageConf, err = a.claimer.GetStorageConfiguration(ctx, edgeDevice)
	} else if storage.ShouldUseExternalConfig(edgeDevice) {
		storageConf, err = a.claimer.GetExternalStorageConfig(ctx, edgeDevice)
	}

	if err == nil && storageConf != nil {
		dc.Configuration.Storage = &models.StorageConfiguration{
			S3: storageConf,
		}
	}

	return err
}

func (a *configurationAssembler) getDeviceMetricsConfiguration(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) (*models.MetricsConfiguration, error) {
	metricsConfig := models.MetricsConfiguration{
		Receiver: a.getMetricsReceiverConfiguration(edgeDevice),
	}

	metricsConfigSpec := edgeDevice.Spec.Metrics
	if metricsConfigSpec == nil {
		return &metricsConfig, nil
	}

	retention := metricsConfigSpec.Retention
	if retention != nil {
		metricsConfig.Retention = &models.MetricsRetention{
			MaxHours: retention.MaxHours,
			MaxMib:   retention.MaxMiB,
		}
	}

	systemMetrics := metricsConfigSpec.SystemMetrics
	if systemMetrics != nil {
		metricsConfig.System = &models.SystemMetricsConfiguration{
			Interval: systemMetrics.Interval,
			Disabled: systemMetrics.Disabled,
		}

		allowListSpec := systemMetrics.AllowList
		if allowListSpec != nil {
			allowList, err := a.allowLists.GenerateFromConfigMap(ctx, allowListSpec.Name, edgeDevice.Namespace)
			if err != nil {
				return nil, err
			}
			metricsConfig.System.AllowList = allowList
		}
	}

	return &metricsConfig, nil
}

func (a *configurationAssembler) toWorkloadList(ctx context.Context, logger logr.Logger, deployments []v1alpha1.EdgeDeployment, device *v1alpha1.EdgeDevice) (models.WorkloadList, error) {
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
			Namespace:     deployment.Namespace,
			Labels:        labels.GetPodmanLabels(deployment.Labels),
			Specification: string(podSpec),
			Data:          data,
			LogCollection: spec.LogCollection,
		}
		authFile, err := a.getAuthFile(ctx, spec.ImageRegistries, deployment.Namespace)
		if err != nil {
			msg := fmt.Sprintf("Auth file secret %s used by deployment %s/%s is missing", spec.ImageRegistries.AuthFileSecret.Name, deployment.Namespace, deployment.Name)
			a.recorder.Event(device, corev1.EventTypeWarning, "Misconfiguration", msg)
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

			if allowListSpec := spec.Metrics.AllowList; allowListSpec != nil {
				allowList, err := a.allowLists.GenerateFromConfigMap(ctx, allowListSpec.Name, deployment.Namespace)
				if err != nil {
					return nil, fmt.Errorf("Cannot get AllowList Metrics Confimap for %v: %v", deployment.Name, err)
				}
				workload.Metrics.AllowList = allowList
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

		configmapList, err := a.configMaps.Fetch(ctx, deployment, device.Namespace)
		if err != nil {
			logger.Error(err, "Faled to fetch configmaps")
			return nil, err
		}
		workload.Configmaps = configmapList
		list = append(list, &workload)
	}
	return list, nil
}

func (a *configurationAssembler) getAuthFile(ctx context.Context, imageRegistries *v1alpha1.ImageRegistriesConfiguration, namespace string) (string, error) {
	if imageRegistries != nil {
		if secretRef := imageRegistries.AuthFileSecret; secretRef != nil {
			authFile, err := a.registryAuthRepository.GetAuthFileFromSecret(ctx, namespace, secretRef.Name)
			if err != nil {
				return "", err
			}
			return authFile, nil
		}
	}
	return "", nil
}

func (a *configurationAssembler) createSecretList(ctx context.Context, logger logr.Logger, deployments []v1alpha1.EdgeDeployment, device *v1alpha1.EdgeDevice) (models.SecretList, error) {
	list := models.SecretList{}

	// create map of secret names and keys
	secretMap := secretMapType{}
	for _, deployment := range deployments {
		podSpec := deployment.Spec.Pod.Spec
		allContainers := append(podSpec.InitContainers, podSpec.Containers...)
		for i := range allContainers {
			extractSecretsFromContainer(&allContainers[i], secretMap)
		}
	}

	// read secrets and add to secrets list
	for name, keys := range secretMap {
		secretObj, err := a.readAndValidateSecret(ctx, name, device.Namespace, keys)
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

func (a *configurationAssembler) readAndValidateSecret(ctx context.Context, secretName, secretNamespace string, secretKeys keyMapType) (*corev1.Secret, error) {
	optional := secretKeys == nil
	secretObj := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace}}
	err := a.client.Get(ctx, client.ObjectKeyFromObject(secretObj), secretObj)
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

func (a *configurationAssembler) getDeviceLogConfig(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice) (map[string]models.LogsCollectionInformation, error) {
	if len(edgeDevice.Spec.LogCollection) == 0 {
		return nil, nil
	}

	res := map[string]models.LogsCollectionInformation{}
	for key, val := range edgeDevice.Spec.LogCollection {
		logConfig := models.LogsCollectionInformation{
			BufferSize: val.BufferSize,
			Kind:       val.Kind,
		}
		if val.SyslogConfig != nil {
			syslogConfig, err := a.getDeviceSyslogLogConfig(ctx, edgeDevice, val)
			if err != nil {
				return nil, err
			}
			logConfig.SyslogConfig = syslogConfig
		}
		res[key] = logConfig
	}
	return res, nil
}

func (a *configurationAssembler) getDeviceSyslogLogConfig(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, val *v1alpha1.LogCollectionConfig) (*models.LogsCollectionInformationSyslogConfig, error) {
	cm := corev1.ConfigMap{}
	err := a.client.Get(ctx,
		client.ObjectKey{Namespace: edgeDevice.Namespace, Name: val.SyslogConfig.Name},
		&cm)
	if err != nil {
		return nil, fmt.Errorf("cannot get syslogconfig from configmap %s: %v", val.SyslogConfig.Name, err)
	}
	proto := "tcp"
	if cmproto, ok := cm.Data["Protocol"]; ok {
		if cmproto == "tcp" || cmproto == "udp" {
			proto = cmproto
		} else {
			return nil, fmt.Errorf("Protocol '%s' is not valid for syslog server", proto)
		}
	}

	return &models.LogsCollectionInformationSyslogConfig{
		Address:  cm.Data["Address"],
		Protocol: proto,
	}, nil
}

func GetDefaultMetricsReceiver() *models.MetricsReceiverConfiguration {
	return &models.MetricsReceiverConfiguration{
		RequestNumSamples: 30000,
		TimeoutSeconds:    10,
	}
}

func (a *configurationAssembler) getMetricsReceiverConfiguration(device *v1alpha1.EdgeDevice) *models.MetricsReceiverConfiguration {
	result := GetDefaultMetricsReceiver()

	if device != nil && device.Spec.Metrics != nil {
		receiverConfig := device.Spec.Metrics.ReceiverConfiguration
		if receiverConfig != nil {
			if receiverConfig.TimeoutSeconds > 0 {
				result.TimeoutSeconds = receiverConfig.TimeoutSeconds
			}
			if receiverConfig.RequestNumSamples > 0 {
				result.RequestNumSamples = receiverConfig.RequestNumSamples
			}
			result.URL = receiverConfig.URL
		}
	}

	return result
}
