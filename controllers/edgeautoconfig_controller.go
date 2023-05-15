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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	mgmtv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"

	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeautoconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
)

const (
	EdgeAutoConfigFinalizer = "edge-auto-config-finalizer"
)

// EdgeAutoConfigReconciler reconciles a EdgeAutoConfig object
type EdgeAutoConfigReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	EdgeAutoConfigRepository edgeautoconfig.Repository
	EdgeDeviceRepository     edgedevice.Repository
	EdgeWorkloadRepository   edgeworkload.Repository
	MaxConcurrentReconciles  int
	AutoApproval             bool
}

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeautoconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeautoconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeautoconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeAutoConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *EdgeAutoConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "edgeautoconfig", req)

	edgeautocfg, err := r.EdgeAutoConfigRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeautocfg.DeletionTimestamp != nil {
		logger.Info("Reconciling", "edgeautoconfig delete")
		// if err := r.removeRelatedEDSR(ctx, edgeDevice); err != nil {
		// 	return ctrl.Result{Requeue: true}, err
		// }
		// return ctrl.Result{}, r.removeFinalizer(ctx, edgeDevice)
	}

	edgeautocfgcpy := edgeautocfg.DeepCopy()
	//get devices which do not have autoconfig workloads set
	edgedevicesstatus := edgeautocfgcpy.Status.EdgeDevices

	for _, edgedevice := range edgedevicesstatus {
		if edgedevice.EdgeDeviceState == managementv1alpha1.EdgeDeviceStatePending {

			//loop through the set images and set to the device
			err = r.createWorkload(ctx, edgeautocfg.Spec.EdgeDeviceWorkloads, edgeautocfg.Name, edgedevice.Name, req.Namespace)
			if err != nil {
				logger.Error(err, "Failed to create auto config workload for device: "+edgedevice.Name)
				return ctrl.Result{}, nil
			}

			//update the autocofig cr status
			edgedevice.EdgeDeviceState = managementv1alpha1.EdgeDeviceStateRunning
			err = r.EdgeAutoConfigRepository.Patch(ctx, edgeautocfg, edgeautocfgcpy)
			if err != nil {
				logger.Error(err, "cannot patch edgeautoconfig status for the workloads")
				return ctrl.Result{Requeue: true}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// create workload
func (r *EdgeAutoConfigReconciler) createWorkload(ctx context.Context, edgeWorkloads []mgmtv1alpha1.EdgeDeviceWorkloads, edgeAutoCfgName, deviceName, nameSpace string) error {

	set_containers := []corev1.Container{}

	for _, edgeWorkload := range edgeWorkloads {
		for i := 0; i < len(edgeWorkload.Containers); i++ {
			container_property := corev1.Container{
				Name:  edgeWorkload.Containers[i].Name,
				Image: edgeWorkload.Containers[i].Image,
			}
			set_containers = append(set_containers, container_property)
		}
	}

	workload_create := &v1alpha1.EdgeWorkload{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      edgeAutoCfgName + deviceName,
			Namespace: nameSpace,
		},
		Spec: v1alpha1.EdgeWorkloadSpec{
			Device: deviceName,
			Type:   "pod",
			Pod: v1alpha1.Pod{
				Spec: corev1.PodSpec{
					Containers: set_containers,
				},
			},
		},
	}

	err := r.EdgeWorkloadRepository.Create(ctx, workload_create)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeAutoConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeAutoConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}
