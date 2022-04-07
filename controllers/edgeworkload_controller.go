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
	"reflect"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/project-flotta/flotta-operator/internal/labels"
	"github.com/project-flotta/flotta-operator/internal/metrics"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/repository/edgeworkload"
	"github.com/project-flotta/flotta-operator/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	managementv1alpha1 "github.com/project-flotta/flotta-operator/api/v1alpha1"
)

const YggdrasilDeviceReferenceFinalizer = "yggdrasil-device-reference-finalizer"

// EdgeWorkloadReconciler reconciles a EdgeWorkload object
type EdgeWorkloadReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	EdgeWorkloadRepository  edgeworkload.Repository
	EdgeDeviceRepository    edgedevice.Repository
	Concurrency             uint
	ExecuteConcurrent       func(uint, ConcurrentFunc, []managementv1alpha1.EdgeDevice) []error
	Metrics                 metrics.Metrics
	MaxConcurrentReconciles int
}

type ConcurrentFunc func([]managementv1alpha1.EdgeDevice) []error

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeworkloads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeworkloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeworkloads/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EdgeWorkload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EdgeWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "edgeWorkload", req)

	// your logic here
	edgeWorkload, err := r.EdgeWorkloadRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if edgeWorkload.DeletionTimestamp == nil && !utils.HasFinalizer(&edgeWorkload.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
		WorkloadCopy := edgeWorkload.DeepCopy()
		WorkloadCopy.Finalizers = []string{YggdrasilDeviceReferenceFinalizer}
		err := r.EdgeWorkloadRepository.Patch(ctx, edgeWorkload, WorkloadCopy)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if edgeWorkload.DeletionTimestamp == nil {
		updated, err := r.updateLabelsFromSelector(ctx, edgeWorkload)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		if updated {
			// if we updated/patched the object then Reconcile will be called for the new version
			// return here in order to avoid executing rest of the code twice
			return ctrl.Result{}, nil
		}
	}

	labelledDevices, err := r.getLabelledEdgeDevices(ctx, edgeWorkload.Name, edgeWorkload.Namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Cannot retrieve labelled Edge Workloads", "edgeWorkload", edgeWorkload.Name, "namespace", edgeWorkload.Namespace)
			return ctrl.Result{Requeue: true}, err
		}
	}
	edgeDevices, err := r.getMatchingEdgeDevices(ctx, edgeWorkload)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Cannot retrieve Edge Workloads")
			return ctrl.Result{Requeue: true}, err
		}
	}

	if edgeWorkload.DeletionTimestamp != nil {
		if utils.HasFinalizer(&edgeWorkload.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
			matchingAndLabelledDevices := merge(edgeDevices, labelledDevices)
			err = r.finalizeRemoval(ctx, matchingAndLabelledDevices, edgeWorkload)
			if err != nil {
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{}, nil
	}

	err = r.addWorkloadsToDevices(ctx, edgeWorkload.Name, edgeDevices)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	err = r.removeWorkloadFromNonMatchingDevices(ctx, edgeWorkload.Name, edgeDevices, labelledDevices)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *EdgeWorkloadReconciler) finalizeRemoval(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeWorkload *managementv1alpha1.EdgeWorkload) error {
	f := func(input []managementv1alpha1.EdgeDevice) []error {
		return r.removeWorkloadFromDevices(ctx, input, edgeWorkload.Name)
	}
	errs := r.executeConcurrent(ctx, f, edgeDevices)
	if len(errs) != 0 {
		return fmt.Errorf(mergeErrorMessages(errs))
	}
	return r.EdgeWorkloadRepository.RemoveFinalizer(ctx, edgeWorkload, YggdrasilDeviceReferenceFinalizer)
}

func (r *EdgeWorkloadReconciler) removeWorkloadFromDevices(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeWorkload string) []error {
	var errs []error
	for _, edgeDevice := range edgeDevices {
		err := r.removeWorkloadFromDevice(ctx, edgeWorkload, edgeDevice)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (r *EdgeWorkloadReconciler) removeWorkloadFromDevice(ctx context.Context, name string, edgeDevice managementv1alpha1.EdgeDevice) error {
	var newWorkloads []managementv1alpha1.Workload
	for _, Workload := range edgeDevice.Status.Workloads {
		if Workload.Name != name {
			newWorkloads = append(newWorkloads, Workload)
		}
	}
	patch := client.MergeFrom(edgeDevice.DeepCopy())
	edgeDevice.Status.Workloads = newWorkloads
	err := r.EdgeDeviceRepository.PatchStatus(ctx, &edgeDevice, &patch)
	if err != nil {
		return err
	}

	deviceCopy := edgeDevice.DeepCopy()
	if deviceCopy.Labels != nil {
		delete(deviceCopy.Labels, labels.WorkloadLabel(name))
		err = r.EdgeDeviceRepository.Patch(ctx, &edgeDevice, deviceCopy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *EdgeWorkloadReconciler) removeWorkloadFromNonMatchingDevices(ctx context.Context, name string, matchingDevices, labelledDevices []managementv1alpha1.EdgeDevice) error {
	matchingDevicesMap := make(map[string]struct{})
	for _, device := range matchingDevices {
		matchingDevicesMap[device.Name] = struct{}{}
	}

	f := func(input []managementv1alpha1.EdgeDevice) []error {
		var errs []error
		for _, device := range input {
			if _, ok := matchingDevicesMap[device.Name]; !ok {
				err := r.removeWorkloadFromDevice(ctx, name, device)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
		}
		return errs
	}

	errs := r.executeConcurrent(ctx, f, labelledDevices)
	if len(errs) != 0 {
		return fmt.Errorf(mergeErrorMessages(errs))
	}

	return nil
}

func (r *EdgeWorkloadReconciler) addWorkloadsToDevices(ctx context.Context, name string, edgeDevices []managementv1alpha1.EdgeDevice) error {
	f := func(input []managementv1alpha1.EdgeDevice) []error {
		var errs []error
		for i := range input {
			edgeDevice := input[i]
			if !hasWorkload(edgeDevice, name) {
				WorkloadStatus := managementv1alpha1.Workload{Name: name, Phase: managementv1alpha1.Deploying}
				patch := client.MergeFrom(edgeDevice.DeepCopy())
				edgeDevice.Status.Workloads = append(edgeDevice.Status.Workloads, WorkloadStatus)
				err := r.EdgeDeviceRepository.PatchStatus(ctx, &edgeDevice, &patch)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			if !hasLabelForWorkload(edgeDevice, name) {
				deviceCopy := edgeDevice.DeepCopy()
				if deviceCopy.Labels == nil {
					deviceCopy.Labels = make(map[string]string)
				}
				deviceCopy.Labels[labels.WorkloadLabel(name)] = "true"
				err := r.EdgeDeviceRepository.Patch(ctx, &edgeDevice, deviceCopy)
				if err != nil {
					errs = append(errs, err)
					continue
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

func (r *EdgeWorkloadReconciler) getMatchingEdgeDevices(ctx context.Context, edgeWorkload *managementv1alpha1.EdgeWorkload) ([]managementv1alpha1.EdgeDevice, error) {
	var edgeDevices []managementv1alpha1.EdgeDevice
	if edgeWorkload.Spec.Device != "" {
		edgeDevice, err := r.EdgeDeviceRepository.Read(ctx, edgeWorkload.Spec.Device, edgeWorkload.Namespace)
		if err != nil {
			return nil, err
		}
		edgeDevices = append(edgeDevices, *edgeDevice)
	} else if edgeWorkload.Spec.DeviceSelector != nil {
		ed, err := r.EdgeDeviceRepository.ListForSelector(ctx, edgeWorkload.Spec.DeviceSelector, edgeWorkload.Namespace)
		if err != nil {
			return nil, err
		}
		edgeDevices = append(edgeDevices, ed...)
	}
	return edgeDevices, nil
}

func (r *EdgeWorkloadReconciler) getLabelledEdgeDevices(ctx context.Context, name, namespace string) ([]managementv1alpha1.EdgeDevice, error) {
	return r.EdgeDeviceRepository.ListForWorkload(ctx, name, namespace)
}

func (r *EdgeWorkloadReconciler) executeConcurrent(ctx context.Context, f ConcurrentFunc, edgeDevices []managementv1alpha1.EdgeDevice) []error {
	var errs []error
	if r.Concurrency == 1 {
		errs = f(edgeDevices)
	} else {
		errs = r.ExecuteConcurrent(r.Concurrency, f, edgeDevices)
	}
	return errs
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeWorkload{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

func ExecuteConcurrent(concurrency uint, f ConcurrentFunc, edgeDevices []managementv1alpha1.EdgeDevice) []error {
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
			returnValues[index] = f(inputs[index])
		}()
	}
	wg.Wait()
	var result []error
	for _, returnValue := range returnValues {
		result = append(result, returnValue...)
	}
	return result
}

func hasLabelForWorkload(edgeDevice managementv1alpha1.EdgeDevice, WorkloadName string) bool {
	_, exists := edgeDevice.Labels[labels.WorkloadLabel(WorkloadName)]
	return exists
}

func hasWorkload(edgeDevice managementv1alpha1.EdgeDevice, name string) bool {
	for _, Workload := range edgeDevice.Status.Workloads {
		if Workload.Name == name {
			return true
		}
	}
	return false
}

func merge(edgeDevices1 []managementv1alpha1.EdgeDevice, edgeDevices2 []managementv1alpha1.EdgeDevice) []managementv1alpha1.EdgeDevice {
	mergedMap := make(map[string]struct{})
	var merged []managementv1alpha1.EdgeDevice
	for _, device := range edgeDevices1 {
		mergedMap[device.Name] = struct{}{}
		merged = append(merged, device)
	}

	for _, device := range edgeDevices2 {
		if _, ok := mergedMap[device.Name]; !ok {
			merged = append(merged, device)
		}
	}

	return merged
}

func mergeErrorMessages(errs []error) string {
	var message string
	for _, err := range errs {
		if message == "" {
			message = err.Error()
			continue
		}
		message += ", " + err.Error()
	}
	return message
}

func splitEdgeDevices(edgeDevices []managementv1alpha1.EdgeDevice, splitSize uint) [][]managementv1alpha1.EdgeDevice {
	if splitSize == 0 {
		return nil
	}
	intX := int(splitSize)
	sLen := len(edgeDevices)
	var result [][]managementv1alpha1.EdgeDevice
	splitLen := sLen / intX
	residueLen := sLen - (splitLen * intX)
	newSliceLen := 0
	for usedLen := 0; usedLen < sLen; usedLen += newSliceLen {
		residueExtra := 0
		if residueLen > 0 {
			residueExtra = 1
			residueLen--
		}
		newSliceLen = splitLen + residueExtra
		newSlice := edgeDevices[usedLen : usedLen+newSliceLen]
		result = append(result, newSlice)
	}
	return result
}

func (r *EdgeWorkloadReconciler) updateLabelsFromSelector(ctx context.Context, edgeWorkload *managementv1alpha1.EdgeWorkload) (bool, error) {
	edgeWorkloadCopy := edgeWorkload.DeepCopy()
	UpdateSelectorLabels(edgeWorkloadCopy)
	newLabels := edgeWorkloadCopy.Labels

	if (len(newLabels) == 0 && edgeWorkload.Labels == nil) || reflect.DeepEqual(newLabels, edgeWorkload.Labels) {
		return false, nil
	}

	err := r.EdgeWorkloadRepository.Patch(ctx, edgeWorkload, edgeWorkloadCopy)
	return true, err
}

func UpdateSelectorLabels(edgeWorkloads ...*managementv1alpha1.EdgeWorkload) {
	for _, edgeWorkload := range edgeWorkloads {
		workloadLabels := edgeWorkload.Labels
		if workloadLabels == nil {
			workloadLabels = map[string]string{}
			edgeWorkload.Labels = workloadLabels
		}
		for label := range workloadLabels {
			if labels.IsSelectorLabel(label) {
				delete(workloadLabels, label)
			}
		}
		if edgeWorkload.Spec.Device != "" {
			selectorLabel := labels.CreateSelectorLabel(labels.DeviceNameLabel)
			workloadLabels[selectorLabel] = edgeWorkload.Spec.Device
		} else {
			labelSelector := edgeWorkload.Spec.DeviceSelector
			if labelSelector != nil {
				for label := range labelSelector.MatchLabels {
					selectorLabel := labels.CreateSelectorLabel(label)
					workloadLabels[selectorLabel] = "true"
				}
				for _, requirement := range labelSelector.MatchExpressions {
					if requirement.Operator == metav1.LabelSelectorOpDoesNotExist {
						selectorLabel := labels.CreateSelectorLabel(labels.DoesNotExistLabel)
						workloadLabels[selectorLabel] = "true"
					} else {
						selectorLabel := labels.CreateSelectorLabel(requirement.Key)
						workloadLabels[selectorLabel] = "true"
					}
				}
			}
		}
	}
}
