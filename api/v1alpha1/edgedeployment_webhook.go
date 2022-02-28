/*
Copyright 2022

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:docs-gen:collapse=Apache License

package v1alpha1

import (
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

//+kubebuilder:docs-gen:collapse=Go imports

func (r *EdgeDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

/*
This marker is responsible for generating a validating webhook manifest.
*/

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-management-project-flotta-io-v1alpha1-edgedeployment,mutating=false,failurePolicy=fail,groups=management.project-flotta.io,resources=edgedeployments,versions=v1alpha1,name=edgedeploymnet.management.project-flotta.io,sideEffects=None,admissionReviewVersions=v1

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeDeployment) ValidateCreate() error {
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeDeployment) ValidateUpdate(old runtime.Object) error {
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeDeployment) ValidateDelete() error {
	return nil
}

func (r *EdgeDeployment) validate() error {
	var notValidPaths []string
	podSpec := r.Spec.Pod.Spec

	containers := append(podSpec.InitContainers, podSpec.Containers...)
	containersNames := make(map[string]struct{})
	for _, container := range containers {
		if container.Lifecycle != nil {
			notValidPaths = append(notValidPaths, containersMsg(container, "lifecycle"))
		}
		if container.LivenessProbe != nil {
			notValidPaths = append(notValidPaths, containersMsg(container, "livenessProbe"))
		}
		if container.ReadinessProbe != nil {
			notValidPaths = append(notValidPaths, containersMsg(container, "readinessProbe"))
		}
		if container.StartupProbe != nil {
			notValidPaths = append(notValidPaths, containersMsg(container, "startupProbe"))
		}
		if len(container.VolumeDevices) != 0 {
			notValidPaths = append(notValidPaths, containersMsg(container, "volumeDevices"))
		}
		if len(container.Resources.Limits) != 0 {
			notValidPaths = append(notValidPaths, containersMsg(container, "resources.limits"))
		}
		if len(container.Resources.Requests) != 0 {
			notValidPaths = append(notValidPaths, containersMsg(container, "resources.requests"))
		}

		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil {
				if envVar.ValueFrom.FieldRef != nil {
					notValidPaths = append(notValidPaths, envMsg(container, envVar, "fieldRef"))
				} else if envVar.ValueFrom.ResourceFieldRef != nil {
					notValidPaths = append(notValidPaths, envMsg(container, envVar, "resourceFieldRef"))
				}
			}
		}

		if _, ok := containersNames[container.Name]; ok {
			return fmt.Errorf("name collisions for containers within the same pod spec are not supported.\n" +
				"container name: '%s' has been reused", container.Name)
		} else {
			containersNames[container.Name] = struct{}{}
		}
	}

	for _, volume := range podSpec.Volumes {
		if volume.HostPath == nil {
			notValidPaths = append(notValidPaths, fmt.Sprintf("volumes[%s]", volume.Name))
		}
	}

	if len(notValidPaths) != 0 {
		return errors.New("the following paths in podSpec are not supported and should be removed: " +
			strings.Join(notValidPaths, ","))
	}

	return nil
}

func containersMsg(container corev1.Container, field string) string {
	return fmt.Sprintf("containers[%s].%s", container.Name, field)
}

func envMsg(container corev1.Container, envVar corev1.EnvVar, field string) string {
	return fmt.Sprintf("containers[%s].env[%s].ValueFrom.%s", container.Name, envVar.Name, field)
}
