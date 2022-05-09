package configmaps

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/k8sclient"
	"github.com/project-flotta/flotta-operator/internal/utils"
	"github.com/project-flotta/flotta-operator/models"
)

//go:generate mockgen -package=configmaps -destination=mock_configmaps.go . ConfigMap
type ConfigMap interface {
	Fetch(ctx context.Context, workload v1alpha1.EdgeWorkload, namespace string) (models.ConfigmapList, error)
}

type configMap struct {
	client k8sclient.K8sClient
}

func NewConfigMap(client k8sclient.K8sClient) ConfigMap {
	return &configMap{client: client}
}

func (cm *configMap) Fetch(ctx context.Context, workload v1alpha1.EdgeWorkload, namespace string) (models.ConfigmapList, error) {
	list := models.ConfigmapList{}

	// create map of configmap names and keys
	cmMap := utils.MapType{}
	podSpec := workload.Spec.Pod.Spec
	allContainers := append(podSpec.InitContainers, podSpec.Containers...)
	for i := range allContainers {
		extractConfigMapsFromEnv(&allContainers[i], cmMap)

		// Extract info also from volumes:
		utils.ExtractInfoFromVolume(podSpec.Volumes, cmMap, func(i interface{}) (bool, *bool, string) {
			volume, ok := i.(corev1.Volume)
			if !ok {
				return false, nil, ""
			}
			if volume.ConfigMap != nil {
				return true, volume.ConfigMap.Optional, volume.ConfigMap.Name
			}
			return false, nil, ""
		})
	}

	// read configmaps and add to configmaps list
	for name, keys := range cmMap {
		configmapObj, err := cm.readAndValidateConfigMap(ctx, name, namespace, keys)
		if err != nil {
			return nil, fmt.Errorf("Can't fetch the configmap %v/%v: %w", name, namespace, err)
		}
		if configmapObj == nil {
			continue
		}
		configmapObj.ObjectMeta = metav1.ObjectMeta{Name: name}
		obj, err := yaml.Marshal(configmapObj)
		if err != nil {
			return nil, err
		}
		list = append(list, string(obj))
	}

	return list, nil
}

func (cm *configMap) readAndValidateConfigMap(ctx context.Context, configmapName, configmapNamespace string, configmapKeys utils.StringSet) (*corev1.ConfigMap, error) {
	configmapObj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configmapName, Namespace: configmapNamespace}}
	err := cm.client.Get(ctx, client.ObjectKeyFromObject(configmapObj), configmapObj)
	if err != nil {
		if errors.IsNotFound(err) && configmapKeys == nil {
			return nil, nil
		}
		return nil, err
	}
	for key := range configmapKeys {
		if configmapObj.Data != nil {
			if _, ok := configmapObj.Data[key]; ok {
				continue
			}
		}
		return nil, fmt.Errorf("missing configmap key. configmap: %s. key: %s. Namespace: %s", configmapName, key, configmapNamespace)
	}
	return configmapObj, nil
}

func extractConfigMapsFromEnv(container *corev1.Container, configmapMap utils.MapType) {
	utils.ExtractInfoFromEnvFrom(container.EnvFrom, configmapMap, func(e interface{}) (bool, *bool, string) {
		env, ok := e.(corev1.EnvFromSource)
		if !ok {
			return false, nil, ""
		}
		if env.ConfigMapRef != nil {
			return true, env.ConfigMapRef.Optional, env.ConfigMapRef.Name
		}
		return false, nil, ""
	})
	utils.ExtractInfoFromEnv(container.Env, configmapMap, func(env corev1.EnvVar) (bool, *bool, string, string) {
		if env.ValueFrom.ConfigMapKeyRef != nil {
			return true, env.ValueFrom.ConfigMapKeyRef.Optional, env.ValueFrom.ConfigMapKeyRef.Name, env.ValueFrom.ConfigMapKeyRef.Key
		}
		return false, nil, "", ""
	})

}
