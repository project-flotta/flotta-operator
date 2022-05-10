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

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/repository/edgeconfig"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EdgeConfigReconciler reconciles a EdgeConfig object
type EdgeConfigReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	EdgeConfigRepository    edgeconfig.Repository
	EdgeDeviceRepository    edgedevice.Repository
	MaxConcurrentReconciles int
}

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *EdgeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "edgeConfig", req)

	// your logic here
	edgeConfig, err := r.EdgeConfigRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeConfig.DeletionTimestamp == nil && !utils.HasFinalizer(&edgeConfig.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
		EdgeConfigCopy := edgeConfig.DeepCopy()
		EdgeConfigCopy.Finalizers = []string{YggdrasilDeviceReferenceFinalizer}
		err := r.EdgeConfigRepository.Patch(ctx, edgeConfig, EdgeConfigCopy)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if edgeConfig.DeletionTimestamp == nil {
		return ctrl.Result{}, nil
	}

	edgeDevices, err := r.EdgeDeviceRepository.ListForEdgeConfig(ctx, edgeConfig.Name, edgeConfig.Namespace)
	if !errors.IsNotFound(err) {
		logger.Error(err, "Cannot retrieve labelled Edge Config", "edgeConfig", edgeConfig.Name, "namespace", edgeConfig.Namespace)
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("EdgeDevice found", "edgeDevices", edgeDevices)
	// *******************
	// TODO : complete
	// *******************
	return ctrl.Result{}, nil
}

func (r *EdgeConfigReconciler) getLabelledEdgeDevices(ctx context.Context, name, namespace string) ([]managementv1alpha1.EdgeDevice, error) {
	return r.EdgeDeviceRepository.ListForWorkload(ctx, name, namespace)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		For(&managementv1alpha1.EdgeConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}
func createPlaybookExecution(edgeConfig *managementv1alpha1.EdgeConfig) managementv1alpha1.PlaybookExecution {
	playbookExec.ObjectMeta.Namespace = edgeConfig.Namespace
	return playbookExec
	playbookExecutionBase := createPlaybookExecution(edgeConfig)

				edgeDevice.Status.PlaybookExecutions = append(edgeDevice.Status.PlaybookExecutions, *playbookExecution)
