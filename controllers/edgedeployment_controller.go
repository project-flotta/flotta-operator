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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
)

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
	edgeDevice, err := r.EdgeDeviceRepository.Read(ctx, edgeDeployment.Spec.Device, edgeDeployment.Namespace)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	err = r.addOwnerReference(ctx, edgeDevice, edgeDeployment)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	if edgeDeployment.Status.Phase == "" {
		edgeDeployment.Status.Phase = managementv1alpha1.Deploying
		edgeDeployment.Status.LastTransitionTime = metav1.Now()
	}
	err = r.EdgeDeploymentRepository.UpdateStatus(ctx, edgeDeployment)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDeployment{}).
		Complete(r)
}

func newEdgeDeviceOwnerReference(ed *managementv1alpha1.EdgeDevice) metav1.OwnerReference {
	blockOwnerDeletion := true
	isController := false
	return metav1.OwnerReference{
		APIVersion:         ed.GroupVersionKind().GroupVersion().String(),
		Kind:               ed.GetObjectKind().GroupVersionKind().Kind,
		Name:               ed.GetName(),
		UID:                ed.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

func (r *EdgeDeploymentReconciler) addOwnerReference(ctx context.Context, owner *managementv1alpha1.EdgeDevice, ownedDeployment *managementv1alpha1.EdgeDeployment) error {
	ownerRefs := ownedDeployment.GetOwnerReferences()
	if ownerRefs == nil {
		ownerRefs = []metav1.OwnerReference{}
	}
	if hasOwnerReference(ownerRefs, owner) {
		return nil
	}
	ownerReference := newEdgeDeviceOwnerReference(owner)

	ownerRefs = append(ownerRefs, ownerReference)

	newEdgeDeployment := ownedDeployment.DeepCopy()
	newEdgeDeployment.SetOwnerReferences(ownerRefs)

	return r.EdgeDeploymentRepository.Patch(ctx, ownedDeployment, newEdgeDeployment)
}

func hasOwnerReference(ownerRefs []metav1.OwnerReference, owner *managementv1alpha1.EdgeDevice) bool {
	for _, ref := range ownerRefs {
		if ref.Name == owner.Name && ref.Kind == owner.Kind && ref.APIVersion == owner.APIVersion {
			return true
		}
	}
	return false
}
