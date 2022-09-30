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

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/playbookexecution"
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
	ExecuteConcurrent           func(context.Context, uint, ConcurrentFunc, []v1alpha1.EdgeDevice) []error
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
	logger.V(1).Info("EdgeConfig Reconcile", "edgeConfig", req)

	edgeConfig, err := r.EdgeConfigRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeConfig.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	edgeDevices, err := r.EdgeDeviceRepository.ListForEdgeConfig(ctx, edgeConfig.Name, edgeConfig.Namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if len(edgeDevices) == 0 {
		return ctrl.Result{}, err
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
		For(&v1alpha1.EdgeConfig{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
		}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func createPlaybookExecutions(edgeConfig *v1alpha1.EdgeConfig) []v1alpha1.PlaybookExecution {
	playbookExecutions := []v1alpha1.PlaybookExecution{}
	for _, peEdgeConfig := range edgeConfig.Spec.EdgePlaybook.Playbooks {
		var playbookExec v1alpha1.PlaybookExecution
		playbookExec.ObjectMeta.Name = edgeConfig.Name
		playbookExec.ObjectMeta.OwnerReferences = []v1.OwnerReference{{
			APIVersion: edgeConfig.APIVersion,
			Kind:       edgeConfig.Kind,
			Name:       edgeConfig.Name,
			UID:        edgeConfig.UID,
		}}
		playbookExec.ObjectMeta.Namespace = edgeConfig.Namespace
		playbookExec.Spec.Playbook = peEdgeConfig
		playbookExec.Spec.ExecutionAttempt = 0
		playbookExecutionStatus := v1alpha1.PlaybookExecutionStatus{
			Conditions: []v1alpha1.PlaybookExecutionCondition{
				{
					Type:   v1alpha1.PlaybookExecutionDeploying,
					Status: v1.ConditionTrue,
				},
			},
		}

		playbookExec.Status = playbookExecutionStatus
		playbookExecutions = append(playbookExecutions, playbookExec)
	}

	return playbookExecutions
}

func (r *EdgeConfigReconciler) addPlaybookExecutionToDevices(ctx context.Context, edgeConfig *v1alpha1.EdgeConfig, edgeDevices []v1alpha1.EdgeDevice) error {
	playbookExecutionBases := createPlaybookExecutions(edgeConfig)
	f := func(ctx context.Context, devices []v1alpha1.EdgeDevice) []error {

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
			edgeDevice := devices[i]
			if !r.hasPlaybookExecution(edgeDevice, edgeDevice.Name+"-"+edgeConfig.Name) {
				patch := client.MergeFrom(edgeDevice.DeepCopy())
				for _, playbookExecutionBases := range playbookExecutionBases {
					playbookExecution := playbookExecutionBases.DeepCopy()
					playbookExecution.Name = edgeDevice.Name + "-" + edgeConfig.Name

					peStatus :=
						v1alpha1.PlaybookExec{Name: playbookExecution.Name,
							PlaybookExecutionStatus: playbookExecution.Status}
					edgeDevice.Status.PlaybookExecutions = append(edgeDevice.Status.PlaybookExecutions, peStatus)

					err := r.PlaybookExecutionRepository.Create(ctx, playbookExecution)
					if err != nil && errors.IsAlreadyExists(err) {
						errs = append(errs, err)
						continue
					}

					if err != nil {
						errs = append(errs, err)
						continue
					}

					err2 := r.EdgeDeviceRepository.PatchStatus(ctx, &edgeDevice, &patch)
					if err2 != nil {
						errs = append(errs, err2)
						continue
					}
				}
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

func (r *EdgeConfigReconciler) hasPlaybookExecution(edgeDevice v1alpha1.EdgeDevice, name string) bool {
	for _, PlaybookExec := range edgeDevice.Status.PlaybookExecutions {
		if PlaybookExec.Name == name {
			return true
		}
	}
	return false
}

func (r *EdgeConfigReconciler) executeConcurrent(ctx context.Context, f ConcurrentFunc, edgeDevices []v1alpha1.EdgeDevice) []error {
	var errs []error
	if r.Concurrency == 1 {
		errs = f(ctx, edgeDevices)
	} else {
		errs = r.ExecuteConcurrent(ctx, r.Concurrency, f, edgeDevices)
	}
	return errs
}

func ExecuteEdgeConfigConcurrent(ctx context.Context, concurrency uint, f ConcurrentFunc, edgeDevices []v1alpha1.EdgeDevice) []error {
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
