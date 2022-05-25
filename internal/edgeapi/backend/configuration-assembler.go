package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-openapi/strfmt"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/labels"
	"github.com/project-flotta/flotta-operator/internal/common/storage"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/k8s"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/configmaps"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/devicemetrics"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/images"
	"github.com/project-flotta/flotta-operator/models"
)

var (
	defaultHeartbeatConfiguration = models.HeartbeatConfiguration{
		HardwareProfile: &models.HardwareProfileConfiguration{},
		PeriodSeconds:   60,
	}
)

type keyMapType = map[string]interface{}
type secretMapType = map[string]keyMapType

type ConfigurationAssembler struct {
	allowLists             devicemetrics.AllowListGenerator
	claimer                *storage.Claimer
	configMaps             configmaps.ConfigMap
	repository             k8s.RepositoryFacade
	recorder               record.EventRecorder
	registryAuthRepository images.RegistryAuthAPI
}

func NewConfigurationAssembler(allowLists devicemetrics.AllowListGenerator,
	claimer *storage.Claimer,
	configMaps configmaps.ConfigMap,
	recorder record.EventRecorder,
	registryAuthRepository images.RegistryAuthAPI,
	repository k8s.RepositoryFacade) *ConfigurationAssembler {
	return &ConfigurationAssembler{
		allowLists:             allowLists,
		claimer:                claimer,
		configMaps:             configMaps,
		repository:             repository,
		recorder:               recorder,
		registryAuthRepository: registryAuthRepository,
	}
}

func (a *ConfigurationAssembler) GetDeviceConfiguration(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, logger *zap.SugaredLogger) (*models.DeviceConfigurationMessage, error) {
	var workloadList models.WorkloadList
	var secretList models.SecretList
	if edgeDevice.DeletionTimestamp == nil {
		var edgeWorkloads []v1alpha1.EdgeWorkload

		for _, workload := range edgeDevice.Status.Workloads {
			edgeWorkload, err := a.repository.GetEdgeWorkload(ctx, workload.Name, edgeDevice.Namespace)
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "cannot retrieve Edge Workloads")
					return nil, err
				}
				continue
			}
			if edgeWorkload.DeletionTimestamp == nil {
				edgeWorkloads = append(edgeWorkloads, *edgeWorkload)
			}
		}
		var err error
		workloadList, err = a.toWorkloadList(ctx, logger, edgeWorkloads, edgeDevice)
		if err != nil {
			return nil, err
		}
		secretList, err = a.createSecretList(ctx, edgeWorkloads, edgeDevice)
		if err != nil {
			logger.Error(err, "failed reading secrets for device workloads")
			return nil, fmt.Errorf("failed reading secrets for device workloads: %w", err)
		}
	}

	dc := models.DeviceConfigurationMessage{
		DeviceID:      edgeDevice.Name,
		Version:       edgeDevice.ResourceVersion,
		Configuration: &models.DeviceConfiguration{},
		Workloads:     workloadList,
		Secrets:       secretList,
	}

	var deviceSet *v1alpha1.EdgeDeviceSet
	if deviceSetName, ok := edgeDevice.Labels["flotta/member-of"]; ok {
		logger.Debug("Device uses EdgeDeviceSet", "edgeDeviceSet", deviceSetName)
		var err error
		deviceSet, err = a.repository.GetEdgeDeviceSet(ctx, deviceSetName, edgeDevice.Namespace)
		if err != nil {
			logger.Info("Cannot load EdgeDeviceSet", "edgeDeviceSet", deviceSetName)
			deviceSet = nil
		}
	}
	dc.Configuration.Heartbeat = getHeartbeatConfiguration(edgeDevice, deviceSet)
	dc.Configuration.Os = getOsConfiguration(edgeDevice, deviceSet)

	var err error
	dc.Configuration.Storage, err = a.getStorageConfiguration(ctx, edgeDevice, deviceSet)
	if err != nil {
		logger.Error(err, "failed to get storage configuration for device")
	}

	dc.Configuration.Metrics, err = a.getDeviceMetricsConfiguration(ctx, edgeDevice, deviceSet)
	if err != nil {
		logger.Error(err, "failed getting device metrics configuration")
		return nil, fmt.Errorf("failed getting device metrics configuration")
	}

	dc.Configuration.LogCollection, err = a.getDeviceLogConfig(ctx, edgeDevice, deviceSet)
	if err != nil {
		logger.Error(err, "failed getting device log configuration")
		return nil, fmt.Errorf("failed getting device log configuration: %w", err)
	}

	return &dc, nil
}

func getOsConfiguration(edgeDevice *v1alpha1.EdgeDevice, deviceSet *v1alpha1.EdgeDeviceSet) *models.OsInformation {
	osInformation := edgeDevice.Spec.OsInformation
	if deviceSet != nil {
		osInformation = deviceSet.Spec.OsInformation
	}

	if osInformation == nil {
		return nil
	}
	return (*models.OsInformation)(osInformation)
}

func getHeartbeatConfiguration(edgeDevice *v1alpha1.EdgeDevice, deviceSet *v1alpha1.EdgeDeviceSet) *models.HeartbeatConfiguration {
	heartbeat := edgeDevice.Spec.Heartbeat
	if deviceSet != nil {
		heartbeat = deviceSet.Spec.Heartbeat
	}

	if heartbeat != nil {
		configuration := models.HeartbeatConfiguration{
			PeriodSeconds: heartbeat.PeriodSeconds,
		}
		if heartbeat.HardwareProfile != nil {
			configuration.HardwareProfile = &models.HardwareProfileConfiguration{
				Include: heartbeat.HardwareProfile.Include,
				Scope:   heartbeat.HardwareProfile.Scope,
			}
		} else {
			configuration.HardwareProfile = defaultHeartbeatConfiguration.HardwareProfile
		}
		return &configuration
	}
	return &defaultHeartbeatConfiguration

}

func (a *ConfigurationAssembler) getStorageConfiguration(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, deviceSet *v1alpha1.EdgeDeviceSet) (*models.StorageConfiguration, error) {
	storageSpec := edgeDevice.Spec.Storage
	if deviceSet != nil {
		storageSpec = deviceSet.Spec.Storage
	}

	var storageConf *models.S3StorageConfiguration
	var err error
	if edgeDevice.Status.DataOBC != nil && len(*edgeDevice.Status.DataOBC) > 0 {
		storageConf, err = a.claimer.GetStorageConfiguration(ctx, edgeDevice)
	} else if storage.ShouldUseExternalConfig(storageSpec) {
		storageConf, err = a.claimer.GetExternalStorageConfig(ctx, edgeDevice.Namespace, storageSpec)
	}

	if err == nil && storageConf != nil {
		return &models.StorageConfiguration{
			S3: storageConf,
		}, nil
	}

	return nil, err
}

func (a *ConfigurationAssembler) getDeviceMetricsConfiguration(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, deviceSet *v1alpha1.EdgeDeviceSet) (*models.MetricsConfiguration, error) {
	metricsConfigSpec := edgeDevice.Spec.Metrics
	if deviceSet != nil {
		metricsConfigSpec = deviceSet.Spec.Metrics
	}

	receiver, err := a.getMetricsReceiverConfiguration(ctx, metricsConfigSpec, edgeDevice.Namespace)
	if err != nil {
		return nil, err
	}

	metricsConfig := models.MetricsConfiguration{
		Receiver: receiver,
	}

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

func (a *ConfigurationAssembler) toWorkloadList(ctx context.Context, logger *zap.SugaredLogger, edgeworkloads []v1alpha1.EdgeWorkload, device *v1alpha1.EdgeDevice) (models.WorkloadList, error) {
	list := models.WorkloadList{}
	for _, edgeworkload := range edgeworkloads {
		if edgeworkload.DeletionTimestamp != nil {
			continue
		}
		spec := edgeworkload.Spec
		podSpec, err := yaml.Marshal(spec.Pod.Spec)
		if err != nil {
			logger.Error(err, "cannot marshal pod specification", "edgeworkload name", edgeworkload.Name)
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
			Name:          edgeworkload.Name,
			Namespace:     edgeworkload.Namespace,
			Labels:        labels.GetPodmanLabels(edgeworkload.Labels),
			Specification: string(podSpec),
			Data:          data,
			LogCollection: spec.LogCollection,
		}
		authFile, err := a.getAuthFile(ctx, spec.ImageRegistries, edgeworkload.Namespace)
		if err != nil {
			msg := fmt.Sprintf("Auth file secret %s used by edgeworkload %s/%s is missing", spec.ImageRegistries.AuthFileSecret.Name, edgeworkload.Namespace, edgeworkload.Name)
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
				allowList, err := a.allowLists.GenerateFromConfigMap(ctx, allowListSpec.Name, edgeworkload.Namespace)
				if err != nil {
					return nil, fmt.Errorf("Cannot get AllowList Metrics Confimap for %v: %w", edgeworkload.Name, err)
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

		configmapList, err := a.configMaps.Fetch(ctx, edgeworkload, device.Namespace)
		if err != nil {
			logger.Error(err, "Faled to fetch configmaps")
			return nil, err
		}
		workload.Configmaps = configmapList
		list = append(list, &workload)
	}
	return list, nil
}

func (a *ConfigurationAssembler) getAuthFile(ctx context.Context, imageRegistries *v1alpha1.ImageRegistriesConfiguration, namespace string) (string, error) {
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

func (a *ConfigurationAssembler) createSecretList(ctx context.Context, workloads []v1alpha1.EdgeWorkload, device *v1alpha1.EdgeDevice) (models.SecretList, error) {
	list := models.SecretList{}

	// create map of secret names and keys
	secretMap := secretMapType{}
	for _, workload := range workloads {
		podSpec := workload.Spec.Pod.Spec
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

func (a *ConfigurationAssembler) readAndValidateSecret(ctx context.Context, secretName, secretNamespace string, secretKeys keyMapType) (*corev1.Secret, error) {
	optional := secretKeys == nil
	secretObj, err := a.repository.GetSecret(ctx, secretName, secretNamespace)
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

func (a *ConfigurationAssembler) getDeviceLogConfig(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, deviceSet *v1alpha1.EdgeDeviceSet) (map[string]models.LogsCollectionInformation, error) {
	logCollection := edgeDevice.Spec.LogCollection
	if deviceSet != nil {
		logCollection = deviceSet.Spec.LogCollection
	}

	if len(logCollection) == 0 {
		return nil, nil
	}

	res := map[string]models.LogsCollectionInformation{}
	for key, val := range logCollection {
		logConfig := models.LogsCollectionInformation{
			BufferSize: val.BufferSize,
			Kind:       val.Kind,
		}
		if val.SyslogConfig != nil {
			syslogConfig, err := a.getDeviceSyslogLogConfig(ctx, edgeDevice.Namespace, val)
			if err != nil {
				return nil, err
			}
			logConfig.SyslogConfig = syslogConfig
		}
		res[key] = logConfig
	}
	return res, nil
}

func (a *ConfigurationAssembler) getDeviceSyslogLogConfig(ctx context.Context, namespace string, val *v1alpha1.LogCollectionConfig) (*models.LogsCollectionInformationSyslogConfig, error) {
	cm, err := a.repository.GetConfigMap(ctx, val.SyslogConfig.Name, namespace)
	if err != nil {
		return nil, fmt.Errorf("cannot get syslogconfig from configmap %s: %w", val.SyslogConfig.Name, err)
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

func (a *ConfigurationAssembler) getMetricsReceiverConfiguration(ctx context.Context, metrics *v1alpha1.MetricsConfiguration, namespace string) (*models.MetricsReceiverConfiguration, error) {
	result := GetDefaultMetricsReceiver()

	if metrics != nil {
		receiverConfig := metrics.ReceiverConfiguration
		if receiverConfig != nil {
			if receiverConfig.TimeoutSeconds > 0 {
				result.TimeoutSeconds = receiverConfig.TimeoutSeconds
			}
			if receiverConfig.RequestNumSamples > 0 {
				result.RequestNumSamples = receiverConfig.RequestNumSamples
			}
			result.URL = receiverConfig.URL

			if result.URL != "" && strings.HasPrefix(result.URL, "https") && receiverConfig.CaSecretName != "" {
				secret, err := a.repository.GetSecret(ctx, receiverConfig.CaSecretName, namespace)

				if err != nil {
					return nil, err
				}

				caBytes := secret.Data["ca.crt"]
				if len(caBytes) == 0 {
					return nil, fmt.Errorf("metrics receiver config - missing key 'ca.crt' in secret %s", secret.Name)
				}

				result.CaCert = string(caBytes)
			}
		}
	}

	return result, nil
}
