/*
Copyright 2021.

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

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var edgedevicelog = logf.Log.WithName("edgedevice-resource")

func (r *EdgeDevice) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-management-project-flotta-io-v1alpha1-edgedevice,mutating=false,failurePolicy=fail,sideEffects=None,groups=management.project-flotta.io,resources=EdgeDevices,verbs=create;update,versions=v1alpha1,name=edgedevice.management.project-flotta.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EdgeDevice{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (d *EdgeDevice) ValidateCreate() error {
	edgedevicelog.Info("validate create", "name", d.Name)
	err := d.validateCreateOrUpdate()
	if err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (d *EdgeDevice) ValidateUpdate(old runtime.Object) error {
	edgedevicelog.Info("validate update", "name", d.Name)
	err := d.validateCreateOrUpdate()
	if err != nil {
		return err
	}
	return nil
}

func (d *EdgeDevice) validateCreateOrUpdate() error {
	if d.Spec.Storage != nil &&
		d.Spec.Storage.S3 != nil &&
		d.Spec.Storage.S3.CreateOBC &&
		(d.Spec.Storage.S3.SecretName != "" || d.Spec.Storage.S3.ConfigMapName != "") {
		return fmt.Errorf("%[1]s=true is invalid when %[2]s or %[3]s values are set.\n"+
			"Either set %[1]s=false or remove the values from %[2]s and %[3]s",
			"spec.storage.s3.createOBC",
			"spec.storage.s3.secretName",
			"spec.storage.s3.configMapName")
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (_ *EdgeDevice) ValidateDelete() error {
	return nil
}
