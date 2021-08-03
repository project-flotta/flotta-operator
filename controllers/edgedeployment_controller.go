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

	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedeployment"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
)

const YggdrasilDeviceReferenceFinalizer = "yggdrasil-device-reference-finalizer"

// EdgeDeploymentReconciler reconciles a EdgeDeployment object
type EdgeDeploymentReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	EdgeDeploymentRepository *edgedeployment.Repository
	EdgeDeviceRepository     *edgedevice.Repository
}

//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EdgeDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "request", req)

	// your logic here
	edgeDeployment, err := r.EdgeDeploymentRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeDeployment.DeletionTimestamp == nil && !utils.HasFinalizer(&edgeDeployment.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
		deploymentCopy := edgeDeployment.DeepCopy()
		deploymentCopy.Finalizers = []string{YggdrasilDeviceReferenceFinalizer}
		err := r.EdgeDeploymentRepository.Patch(ctx, edgeDeployment, deploymentCopy)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	edgeDevices, err := r.getMatchingEdgeDevices(ctx, edgeDeployment)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Cannot retrieve Edge Deployments")
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info("No Devices found")
	}

	if edgeDeployment.DeletionTimestamp != nil {
		if utils.HasFinalizer(&edgeDeployment.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
			err = r.finalizeRemoval(ctx, edgeDevices, edgeDeployment)
			if err != nil {
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{}, nil
	}

	err = r.addDeploymentsToDevices(ctx, edgeDevices, edgeDeployment)
	// TODO: Label Device with the EdgeDeployment name to allow easy retrieval and bookkeeping
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *EdgeDeploymentReconciler) finalizeRemoval(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeDeployment *managementv1alpha1.EdgeDeployment) error {
	for _, edgeDevice := range edgeDevices {
		var newDeployemnts []managementv1alpha1.Deployment
		for _, deployment := range edgeDevice.Status.Deployments {
			if deployment.Name != edgeDeployment.Name {
				newDeployemnts = append(newDeployemnts, deployment)
			}
		}
		edgeDevice.Status.Deployments = newDeployemnts
		err := r.EdgeDeviceRepository.UpdateStatus(ctx, &edgeDevice)
		if err != nil {
			return err
		}
	}
	return r.EdgeDeploymentRepository.RemoveFinalizer(ctx, edgeDeployment, YggdrasilDeviceReferenceFinalizer)

}

func (r *EdgeDeploymentReconciler) addDeploymentsToDevices(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeDeployment *managementv1alpha1.EdgeDeployment) error {
	for _, edgeDevice := range edgeDevices {
		for _, deployment := range edgeDevice.Status.Deployments {
			if deployment.Name == edgeDeployment.Name {
				return nil
			}
		}
		deploymentStatus := managementv1alpha1.Deployment{Name: edgeDeployment.Name, Phase: managementv1alpha1.Deploying}
		edgeDevice.Status.Deployments = append(edgeDevice.Status.Deployments, deploymentStatus)
		err := r.EdgeDeviceRepository.UpdateStatus(ctx, &edgeDevice)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *EdgeDeploymentReconciler) getMatchingEdgeDevices(ctx context.Context, edgeDeployment *managementv1alpha1.EdgeDeployment) ([]managementv1alpha1.EdgeDevice, error) {
	var edgeDevices []managementv1alpha1.EdgeDevice
	if edgeDeployment.Spec.Device != "" {
		edgeDevice, err := r.EdgeDeviceRepository.Read(ctx, edgeDeployment.Spec.Device, edgeDeployment.Namespace)
		if err != nil {
			return nil, err
		}
		edgeDevices = append(edgeDevices, *edgeDevice)
	} else if edgeDeployment.Spec.DeviceSelector != nil {
		ed, err := r.EdgeDeviceRepository.ListForSelector(ctx, edgeDeployment.Spec.DeviceSelector, edgeDeployment.Namespace)
		if err != nil {
			return nil, err
		}
		edgeDevices = append(edgeDevices, ed...)
	}
	return edgeDevices, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDeployment{}).
		Complete(r)
}
