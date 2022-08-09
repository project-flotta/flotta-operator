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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	ignition "github.com/project-flotta/flotta-device-configuration/pkg/ignition"
)

//+kubebuilder:docs-gen:collapse=Go imports

func (r *EdgeConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

/*
This marker is responsible for generating a validating webhook manifest.
*/

//+kubebuilder:webhook:verbs=create;update,path=/validate-management-project-flotta-io-v1alpha1-EdgeConfig,mutating=false,failurePolicy=fail,groups=management.project-flotta.io,resources=EdgeConfig,versions=v1alpha1,name=edgeconfig.management.project-flotta.io,sideEffects=None,admissionReviewVersions=v1

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeConfig) ValidateCreate() error {
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeConfig) ValidateUpdate(old runtime.Object) error {
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EdgeConfig) ValidateDelete() error {
	return nil
}

func (r *EdgeConfig) validate() error {
	if len(r.Spec.IgnitionConfig) == 0 {
		return nil
	}

	_, err := ignition.ParseConfig(string(r.Spec.IgnitionConfig))
	return err
}
