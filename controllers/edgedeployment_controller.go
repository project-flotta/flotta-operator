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
	"github.com/project-flotta/flotta-operator/internal/repository/edgedeployment"
	"github.com/project-flotta/flotta-operator/internal/repository/edgedevice"
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

// EdgeDeploymentReconciler reconciles a EdgeDeployment object
type EdgeDeploymentReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	EdgeDeploymentRepository edgedeployment.Repository
	EdgeDeviceRepository     edgedevice.Repository
	Concurrency              uint
	ExecuteConcurrent        func(uint, ConcurrentFunc, []managementv1alpha1.EdgeDevice) []error
	Metrics                  metrics.Metrics
	MaxConcurrentReconciles  int
}

type ConcurrentFunc func([]managementv1alpha1.EdgeDevice) []error

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedeployments/finalizers,verbs=update

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
	logger.Info("Reconciling", "edgeDeployment", req)

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

	if edgeDeployment.DeletionTimestamp == nil {
		updated, err := r.updateLabelsFromSelector(ctx, edgeDeployment)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		if updated {
			// if we updated/patched the object then Reconcile will be called for the new version
			// return here in order to avoid executing rest of the code twice
			return ctrl.Result{}, nil
		}
	}

	labelledDevices, err := r.getLabelledEdgeDevices(ctx, edgeDeployment.Name, edgeDeployment.Namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Cannot retrieve labelled Edge Deployments", "edgeDeployment", edgeDeployment.Name, "namespace", edgeDeployment.Namespace)
			return ctrl.Result{Requeue: true}, err
		}
	}
	edgeDevices, err := r.getMatchingEdgeDevices(ctx, edgeDeployment)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Cannot retrieve Edge Deployments")
			return ctrl.Result{Requeue: true}, err
		}
	}

	if edgeDeployment.DeletionTimestamp != nil {
		if utils.HasFinalizer(&edgeDeployment.ObjectMeta, YggdrasilDeviceReferenceFinalizer) {
			matchingAndLabelledDevices := merge(edgeDevices, labelledDevices)
			err = r.finalizeRemoval(ctx, matchingAndLabelledDevices, edgeDeployment)
			if err != nil {
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{}, nil
	}

	err = r.addDeploymentsToDevices(ctx, edgeDeployment.Name, edgeDevices)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	err = r.removeDeploymentFromNonMatchingDevices(ctx, edgeDeployment.Name, edgeDevices, labelledDevices)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *EdgeDeploymentReconciler) finalizeRemoval(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeDeployment *managementv1alpha1.EdgeDeployment) error {
	f := func(input []managementv1alpha1.EdgeDevice) []error {
		return r.removeDeploymentFromDevices(ctx, input, edgeDeployment.Name)
	}
	errs := r.executeConcurrent(ctx, f, edgeDevices)
	if len(errs) != 0 {
		return fmt.Errorf(mergeErrorMessages(errs))
	}
	return r.EdgeDeploymentRepository.RemoveFinalizer(ctx, edgeDeployment, YggdrasilDeviceReferenceFinalizer)
}

func (r *EdgeDeploymentReconciler) removeDeploymentFromDevices(ctx context.Context, edgeDevices []managementv1alpha1.EdgeDevice, edgeDeployment string) []error {
	var errs []error
	for _, edgeDevice := range edgeDevices {
		err := r.removeDeploymentFromDevice(ctx, edgeDeployment, edgeDevice)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (r *EdgeDeploymentReconciler) removeDeploymentFromDevice(ctx context.Context, name string, edgeDevice managementv1alpha1.EdgeDevice) error {
	var newDeployments []managementv1alpha1.Deployment
	for _, deployment := range edgeDevice.Status.Deployments {
		if deployment.Name != name {
			newDeployments = append(newDeployments, deployment)
		}
	}
	patch := client.MergeFrom(edgeDevice.DeepCopy())
	edgeDevice.Status.Deployments = newDeployments
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

func (r *EdgeDeploymentReconciler) removeDeploymentFromNonMatchingDevices(ctx context.Context, name string, matchingDevices, labelledDevices []managementv1alpha1.EdgeDevice) error {
	matchingDevicesMap := make(map[string]struct{})
	for _, device := range matchingDevices {
		matchingDevicesMap[device.Name] = struct{}{}
	}

	f := func(input []managementv1alpha1.EdgeDevice) []error {
		var errs []error
		for _, device := range input {
			if _, ok := matchingDevicesMap[device.Name]; !ok {
				err := r.removeDeploymentFromDevice(ctx, name, device)
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

func (r *EdgeDeploymentReconciler) addDeploymentsToDevices(ctx context.Context, name string, edgeDevices []managementv1alpha1.EdgeDevice) error {
	f := func(input []managementv1alpha1.EdgeDevice) []error {
		var errs []error
		for i := range input {
			edgeDevice := input[i]
			if !hasDeployment(edgeDevice, name) {
				deploymentStatus := managementv1alpha1.Deployment{Name: name, Phase: managementv1alpha1.Deploying}
				patch := client.MergeFrom(edgeDevice.DeepCopy())
				edgeDevice.Status.Deployments = append(edgeDevice.Status.Deployments, deploymentStatus)
				err := r.EdgeDeviceRepository.PatchStatus(ctx, &edgeDevice, &patch)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			if !hasLabelForDeployment(edgeDevice, name) {
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

func (r *EdgeDeploymentReconciler) getLabelledEdgeDevices(ctx context.Context, name, namespace string) ([]managementv1alpha1.EdgeDevice, error) {
	selector := metav1.LabelSelector{MatchLabels: map[string]string{labels.WorkloadLabel(name): "true"}}
	return r.EdgeDeviceRepository.ListForSelector(ctx, &selector, namespace)
}

func (r *EdgeDeploymentReconciler) executeConcurrent(ctx context.Context, f ConcurrentFunc, edgeDevices []managementv1alpha1.EdgeDevice) []error {
	var errs []error
	if r.Concurrency == 1 {
		errs = f(edgeDevices)
	} else {
		errs = r.ExecuteConcurrent(r.Concurrency, f, edgeDevices)
	}
	return errs
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDeployment{}).
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

func hasLabelForDeployment(edgeDevice managementv1alpha1.EdgeDevice, deploymentName string) bool {
	_, exists := edgeDevice.Labels[labels.WorkloadLabel(deploymentName)]
	return exists
}

func hasDeployment(edgeDevice managementv1alpha1.EdgeDevice, name string) bool {
	for _, deployment := range edgeDevice.Status.Deployments {
		if deployment.Name == name {
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

func (r *EdgeDeploymentReconciler) updateLabelsFromSelector(ctx context.Context, edgeDeployment *managementv1alpha1.EdgeDeployment) (bool, error) {
	edgeDeploymentCopy := edgeDeployment.DeepCopy()
	UpdateSelectorLabels(edgeDeploymentCopy)
	newLabels := edgeDeploymentCopy.Labels

	if (len(newLabels) == 0 && edgeDeployment.Labels == nil) || reflect.DeepEqual(newLabels, edgeDeployment.Labels) {
		return false, nil
	}

	err := r.EdgeDeploymentRepository.Patch(ctx, edgeDeployment, edgeDeploymentCopy)
	return true, err
}

func UpdateSelectorLabels(edgeDeployments ...*managementv1alpha1.EdgeDeployment) {
	for _, edgeDeployment := range edgeDeployments {
		deploymentLabels := edgeDeployment.Labels
		if deploymentLabels == nil {
			deploymentLabels = map[string]string{}
			edgeDeployment.Labels = deploymentLabels
		}
		for label := range deploymentLabels {
			if labels.IsSelectorLabel(label) {
				delete(deploymentLabels, label)
			}
		}
		if edgeDeployment.Spec.Device != "" {
			selectorLabel := labels.CreateSelectorLabel(labels.DeviceNameLabel)
			deploymentLabels[selectorLabel] = edgeDeployment.Spec.Device
		} else {
			labelSelector := edgeDeployment.Spec.DeviceSelector
			if labelSelector != nil {
				for label := range labelSelector.MatchLabels {
					selectorLabel := labels.CreateSelectorLabel(label)
					deploymentLabels[selectorLabel] = "true"
				}
				for _, requirement := range labelSelector.MatchExpressions {
					if requirement.Operator == metav1.LabelSelectorOpDoesNotExist {
						selectorLabel := labels.CreateSelectorLabel(labels.DoesNotExistLabel)
						deploymentLabels[selectorLabel] = "true"
					} else {
						selectorLabel := labels.CreateSelectorLabel(requirement.Key)
						deploymentLabels[selectorLabel] = "true"
					}
				}
			}
		}
	}
}
