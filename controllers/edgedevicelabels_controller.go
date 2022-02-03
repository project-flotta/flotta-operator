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

package controllers

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	flottalabels "github.com/project-flotta/flotta-operator/internal/labels"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EdgeDeviceReconciler reconciles a EdgeDevice object
type EdgeDeviceLabelsReconciler struct {
	EdgeDeviceRepository     edgedevice.Repository
	EdgeDeploymentRepository edgedeployment.Repository
	MaxConcurrentReconciles  int
}

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevices,verbs=get;watch;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevices/status,verbs=get;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedeployments,verbs=list

func (r *EdgeDeviceLabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("labels")

	logger.Info("Reconciling")

	edgeDevice, err := r.EdgeDeviceRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeDevice.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	err = r.updateDeployments(ctx, edgeDevice)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeviceLabelsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDevice{}, builder.WithPredicates(predicate.LabelChangedPredicate{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func (r *EdgeDeviceLabelsReconciler) updateDeployments(ctx context.Context, device *managementv1alpha1.EdgeDevice) error {
	// create selector labels
	selectorLabels := createSelectorLabelsMap(device)

	// read deployments matching the labels and match to device
	selectedDeployments := map[string]bool{} // each deployment we read is here. the value is true if the deployment matches the device

	for selectorLabel, labelValue := range selectorLabels {
		deployments, err := r.EdgeDeploymentRepository.ListByLabel(ctx, selectorLabel, labelValue)
		if err != nil {
			return err
		}

		for i := range deployments {
			deployment := deployments[i]
			if _, ok := selectedDeployments[deployment.Name]; ok {
				continue
			}
			match, err := isDeploymentMatchDevice(&deployment, device)
			if err != nil {
				return err
			}
			selectedDeployments[deployment.Name] = match
		}
	}

	// diff device deployments and matched deployments. update device if necessary
	updatedDevice := createUpdatedDevice(selectedDeployments, device)
	if updatedDevice != nil {
		patch := client.MergeFrom(device)
		err := r.EdgeDeviceRepository.PatchStatus(ctx, updatedDevice, &patch)
		if err != nil {
			return err
		}
		err = r.EdgeDeviceRepository.Patch(ctx, device, updatedDevice)
		if err != nil {
			return err
		}
	}

	return nil
}

func isDeploymentMatchDevice(deployment *managementv1alpha1.EdgeDeployment, device *managementv1alpha1.EdgeDevice) (bool, error) {
	if deployment.Spec.Device == device.Name {
		return true, nil
	} else if deployment.Spec.DeviceSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.DeviceSelector)
		if err != nil {
			return false, err
		}
		return selector.Matches(labels.Set(device.Labels)), nil
	}
	return false, nil
}

func createSelectorLabelsMap(device *managementv1alpha1.EdgeDevice) map[string]string {
	result := map[string]string{
		flottalabels.CreateSelectorLabel(flottalabels.DeviceNameLabel):   device.Name,
		flottalabels.CreateSelectorLabel(flottalabels.DoesNotExistLabel): "true",
	}

	for deviceLabel := range device.Labels {
		if flottalabels.IsWorkloadLabel(deviceLabel) {
			continue
		}
		selectorLabel := flottalabels.CreateSelectorLabel(deviceLabel)
		result[selectorLabel] = "true"
	}

	return result
}

func createUpdatedDevice(selectedDeployments map[string]bool, device *managementv1alpha1.EdgeDevice) *managementv1alpha1.EdgeDevice {
	// prepare a copy of the device for modifying
	deviceCopy := device.DeepCopy()
	deviceCopy.Status.Deployments = nil
	if deviceCopy.Labels == nil {
		deviceCopy.Labels = map[string]string{}
	}
	deviceUpdated := false

	// go over device deployments
	// if exist then remove from map
	// if not exist in map then remove deployment and label
	// go over map and add the remaining deployments
	for _, deployment := range device.Status.Deployments {
		if match, ok := selectedDeployments[deployment.Name]; ok && match {
			deviceCopy.Status.Deployments = append(deviceCopy.Status.Deployments, deployment)
			delete(selectedDeployments, deployment.Name)
		} else {
			delete(deviceCopy.Labels, flottalabels.WorkloadLabel(deployment.Name))
			deviceUpdated = true
		}
	}

	for name, match := range selectedDeployments {
		if !match {
			continue
		}
		deviceUpdated = true
		deviceCopy.Status.Deployments = append(deviceCopy.Status.Deployments, managementv1alpha1.Deployment{
			Name:  name,
			Phase: managementv1alpha1.Deploying,
		})
		deviceCopy.Labels[flottalabels.WorkloadLabel(name)] = "true"
	}

	if deviceUpdated {
		return deviceCopy
	}

	return nil
}
