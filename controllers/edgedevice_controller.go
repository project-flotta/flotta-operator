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
	"time"

	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/internal/storage"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	managementv1alpha1 "github.com/jakub-dzon/k4e-operator/api/v1alpha1"
)

// EdgeDeviceReconciler reconciles a EdgeDevice object
type EdgeDeviceReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	EdgeDeviceRepository *edgedevice.CRRepository
	ObcAutoCreate        bool
	Claimer              *storage.Claimer
}

//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedevices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedevices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.k4e.io,resources=edgedevices/finalizers,verbs=update
//+kubebuilder:rbac:groups=objectbucket.io,resources=objectbucketclaims,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeDevice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EdgeDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "request", req)

	edgeDevice, err := r.EdgeDeviceRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("Reconciling", "edgeDevice", edgeDevice)

	if !r.ObcAutoCreate {
		return ctrl.Result{}, nil
	}
	// create object bucket claim for edge-device
	if edgeDevice.Status.DataOBC == nil || len(*edgeDevice.Status.DataOBC) == 0 {
		obc, err := r.createOrGetObc(ctx, edgeDevice)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		err = r.addObcReference(ctx, edgeDevice, obc.Name)
		if err != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *EdgeDeviceReconciler) createOrGetObc(ctx context.Context, edgeDevice *managementv1alpha1.EdgeDevice) (*obv1.ObjectBucketClaim, error) {
	obc, err := r.Claimer.GetClaim(ctx, edgeDevice.Name, edgeDevice.Namespace)
	if err == nil {
		return obc, err
	}

	logger := log.FromContext(ctx)
	if errors.IsNotFound(err) {
		logger.Info("Failed to find an existing OBC for the device. Creating new OBC", "edgeDevice", edgeDevice)
		obc, err = r.Claimer.CreateClaim(ctx, edgeDevice)
		if err != nil {
			logger.Error(err, "Cannot create object bucket claim for the device", "EdgeDevice Name", edgeDevice.Name, "EdgeDevice Namespace", edgeDevice.Namespace)
			return nil, err
		}
		return obc, nil
	}

	logger.Error(err, "Failed to get OBC for the device", "edgeDevice", edgeDevice)
	return nil, err
}

func (r *EdgeDeviceReconciler) addObcReference(ctx context.Context, edgeDevice *managementv1alpha1.EdgeDevice, obcName string) error {
	patch := client.MergeFrom(edgeDevice.DeepCopy())
	edgeDevice.Status.DataOBC = &obcName
	return r.EdgeDeviceRepository.PatchStatus(ctx, edgeDevice, &patch)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDevice{}).
		Complete(r)
}
