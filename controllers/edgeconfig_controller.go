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
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/playbookexecution"
	"github.com/project-flotta/flotta-operator/internal/common/utils"
)

// EdgeConfigReconciler reconciles a EdgeConfig object
type EdgeConfigReconciler struct {
	client.Client
	Scheme                      *runtime.Scheme
	EdgeConfigRepository        edgeconfig.Repository
	EdgeDeviceRepository        edgedevice.Repository
	PlaybookExecutionRepository playbookexecution.Repository
	Concurrency                 uint
	MaxConcurrentReconciles     int
	ExecuteConcurrent           func(context.Context, uint, ConcurrentFunc, []managementv1alpha1.EdgeDevice) []error
}

var logger = log.FromContext(context.TODO())

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
	logger.V(1).Info("EdgeConfig Reconcile", "edgeConfig", req)

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

	edgeDevices, err := r.EdgeDeviceRepository.ListForEdgeConfig(ctx, edgeConfig.Name, edgeConfig.Namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, err
		}
	}
	err = r.addPlaybookExecutionToDevices(ctx, edgeConfig, edgeDevices)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&managementv1alpha1.EdgeConfig{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
		}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func createPlaybookExecution(edgeConfig *managementv1alpha1.EdgeConfig) managementv1alpha1.PlaybookExecution {
	var playbookExec managementv1alpha1.PlaybookExecution
	playbookExec.ObjectMeta.Name = edgeConfig.Name
	playbookExec.ObjectMeta.Namespace = edgeConfig.Namespace
	playbookExec.Spec.Playbook = edgeConfig.Spec.EdgePlaybook.Playbooks[0] //TODO Iterate over the playbooks
	playbookExec.Spec.ExecutionAttempt = 0
	playbookExecutionStatus := managementv1alpha1.PlaybookExecutionStatus{}
	playbookExecutionStatus.Conditions = append(playbookExecutionStatus.Conditions, managementv1alpha1.PlaybookExecutionCondition{Type: managementv1alpha1.PlaybookExecutionDeploying, Status: v1.ConditionTrue})
	playbookExec.Status = playbookExecutionStatus
	return playbookExec
}

func (r *EdgeConfigReconciler) addPlaybookExecutionToDevices(ctx context.Context, edgeConfig *managementv1alpha1.EdgeConfig, edgeDevices []managementv1alpha1.EdgeDevice) error {
	playbookExecutionBase := createPlaybookExecution(edgeConfig)
	f := func(ctx context.Context, devices []managementv1alpha1.EdgeDevice) []error {

		var errs []error
		// access the item in the iterable directly instead of using the iterator variable
		// to fix implicit memory aliasing in for loop
		for i := range devices {
			select {
			case <-ctx.Done():
				errs = append(errs, fmt.Errorf("context canceled: %w", ctx.Err()))
				return errs
			default:
			}
			if !r.hasPlaybookExecution(devices[i], edgeConfig.Name) {
				playbookExecution := playbookExecutionBase.DeepCopy()
				playbookExecution.Name = devices[i].Name + "-" + edgeConfig.Name

				peStatus := managementv1alpha1.PlaybookExec{Name: playbookExecution.Name}
				devices[i].Status.PlaybookExecutions = append(devices[i].Status.PlaybookExecutions, peStatus)
				err := r.PlaybookExecutionRepository.Create(ctx, playbookExecution)
				if err != nil {
					if errors.IsAlreadyExists(err) {
						continue
					}
					errs = append(errs, err)
					continue
				}
				patch := client.MergeFrom(devices[i].DeepCopy())
				err = r.EdgeDeviceRepository.PatchStatus(ctx, &devices[i], &patch)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			} else {
				logger.Info("Edge Device has already a playbookExecution", "edgeDevice", devices[i], "edgeConfig", edgeConfig)
			}
		}

		return errs
	}

	errs := r.executeConcurrent(ctx, f, edgeDevices)
	if len(errs) != 0 {
		return fmt.Errorf(mergeErrorMessages(errs))
	}
	return nil
}

func (r *EdgeConfigReconciler) hasPlaybookExecution(edgeDevice managementv1alpha1.EdgeDevice, name string) bool {
	for _, PlaybookExec := range edgeDevice.Status.PlaybookExecutions {
		if PlaybookExec.Name == name {
			return true
		}
	}
	return false
}

func (r *EdgeConfigReconciler) executeConcurrent(ctx context.Context, f ConcurrentFunc, edgeDevices []managementv1alpha1.EdgeDevice) []error {
	var errs []error
	if r.Concurrency == 1 {
		errs = f(ctx, edgeDevices)
	} else {
		errs = r.ExecuteConcurrent(ctx, r.Concurrency, f, edgeDevices)
	}
	return errs
}

func ExecuteEdgeConfigConcurrent(ctx context.Context, concurrency uint, f ConcurrentFunc, edgeDevices []managementv1alpha1.EdgeDevice) []error {
	if len(edgeDevices) == 0 || concurrency == 0 {
		return nil
	}
	inputs := splitEdgeDevices(edgeDevices, concurrency)
	nInputs := len(inputs)
	returnValues := make([][]error, nInputs)
	var wg sync.WaitGroup
	wg.Add(nInputs)
	for i := 0; i < nInputs; i++ {
		index := i
		go func() {
			defer wg.Done()
			returnValues[index] = f(ctx, inputs[index])
		}()
	}
	wg.Wait()
	var result []error
	for _, returnValue := range returnValues {
		result = append(result, returnValue...)
	}
	return result
}
