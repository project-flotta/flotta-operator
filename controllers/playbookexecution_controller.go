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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/playbookexecution"
	"github.com/project-flotta/flotta-operator/internal/common/utils"
)

// PlaybookExecutionReconciler reconciles a PlaybookExecution object
type PlaybookExecutionReconciler struct {
	client.Client
	Scheme                      *runtime.Scheme
	EdgeDeviceRepository        edgedevice.Repository
	PlaybookExecutionRepository playbookexecution.Repository
}

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=playbookexecutions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=playbookexecutions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=playbookexecutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PlaybookExecution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PlaybookExecutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "playbookExecution", req)

	playbookExec, err := r.PlaybookExecutionRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}
	if playbookExec.DeletionTimestamp == nil && !utils.HasFinalizer(&playbookExec.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
		PlaybookExecCopy := playbookExec.DeepCopy()
		PlaybookExecCopy.Finalizers = []string{YggdrasilDeviceReferenceFinalizer}
		err := r.PlaybookExecutionRepository.Patch(ctx, playbookExec, PlaybookExecCopy)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	edgeDevice, err := r.EdgeDeviceRepository.ReadForPlaybookExecution(ctx, playbookExec.Name, req.Namespace) //TODO use edgeDevice
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("edgeDevice found", "edgeDevice", edgeDevice)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlaybookExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PlaybookExecution{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
		}).
		Complete(r)
}
