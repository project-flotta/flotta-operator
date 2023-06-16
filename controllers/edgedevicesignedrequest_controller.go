package controllers

import (
	"context"
	"time"

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
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeautoconfig"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevicesignedrequest"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgeworkload"
)

type EdgeDeviceSignedRequestReconciler struct {
	client.Client
	Scheme                            *runtime.Scheme
	EdgedeviceSignedRequestRepository edgedevicesignedrequest.Repository
	EdgeDeviceRepository              edgedevice.Repository
	EdgeAutoConfigRepository          edgeautoconfig.Repository
	EdgeWorkloadRepository            edgeworkload.Repository
	MaxConcurrentReconciles           int
	AutoApproval                      bool
}

//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeautoconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgeautoconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevicesignedrequest,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=management.project-flotta.io,resources=edgedevicesignedrequest/status,verbs=get;update;patch

// Reconcile each edgedevicesignedrequest
func (r *EdgeDeviceSignedRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "edgedeviceSignedRequest", req)

	edsr, err := r.EdgedeviceSignedRequestRepository.Read(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	if r.AutoApproval && !edsr.Spec.Approved {
		newEDSR := edsr.DeepCopy()
		newEDSR.Spec.Approved = true
		err := r.EdgedeviceSignedRequestRepository.Patch(ctx, edsr, newEDSR)
		if err != nil {
			logger.Error(err, "cannot set device request to auto-approved.")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !edsr.Spec.Approved {
		if IsPending(edsr) && len(edsr.Status.Conditions) > 0 {
			return ctrl.Result{}, nil
		}
		if err := r.markStatusPending(ctx, edsr); err != nil {
			logger.Error(err, "cannot patch status to pending")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	_, err = r.EdgeDeviceRepository.Read(ctx, edsr.Name, edsr.Spec.TargetNamespace)
	if err == nil {
		// device is already created
		if err := r.markStatusApproved(ctx, edsr); err != nil {
			logger.Error(err, "cannot patch status to approved")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	now := v1.Now()
	device := &v1alpha1.EdgeDevice{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      edsr.Name,
			Namespace: edsr.Spec.TargetNamespace,
			Labels: map[string]string{
				v1alpha1.EdgeDeviceSignedRequestLabelName: v1alpha1.EdgeDeviceSignedRequestLabelValue,
			},
			Finalizers: []string{
				DeviceFinalizer,
			},
		},
		Spec: managementv1alpha1.EdgeDeviceSpec{
			RequestTime: &now,
		},
	}

	if edsr.Spec.TargetSet != "" {
		device.ObjectMeta.Labels[v1alpha1.EdgedeviceSetLabel] = edsr.Spec.TargetSet
	}

	err = r.EdgeDeviceRepository.Create(ctx, device)
	if err != nil {
		logger.Error(err, "cannot create edgedevice")
		return ctrl.Result{Requeue: true}, err
	}

	if err := r.markStatusApproved(ctx, edsr); err != nil {
		logger.Error(err, "cannot patch status to approved")
		return ctrl.Result{Requeue: true}, err
	}

	//New code
	//Deploy workloads to device configured in autoconfig
	//check if auto config is there

	edgeAutoCfgList, err := r.EdgeAutoConfigRepository.ListByNamespace(ctx, edsr.Spec.TargetNamespace)
	if err != nil {
		logger.Error(err, " AUTOCONFIG LISTING ERROR ")
		return ctrl.Result{}, nil
	}

	for i := range edgeAutoCfgList {

		autocfgcpy := edgeAutoCfgList[i].DeepCopy()

		edsrDeviceFeatures := edsr.Spec.Features
		AutoConfigPreferredDevice := autocfgcpy.Spec.EdgeDeviceProperties

		isMatchingDevice := r.compareDeviceFeaturesPreferred(edsrDeviceFeatures, AutoConfigPreferredDevice)
		if isMatchingDevice {
			logger.Info("deviceMatch", "Devices matches preferred AutoConfig ", AutoConfigPreferredDevice)
			//update the autoconfig CR with the new device being registered
			deviceName := edsr.Name
			if err := r.patchEdgeAutoConfigStatus(ctx, autocfgcpy, deviceName, v1alpha1.EdgeDeviceStatePending); err != nil {
				logger.Error(err, "cannot patch status to EDGEAUTOCONFIG ", autocfgcpy.Name)
				return ctrl.Result{Requeue: true}, err
			}
			logger.Info("AutoConfig", "AutoConfig Status Patched with new device", AutoConfigPreferredDevice)

			return ctrl.Result{}, nil
		}
		logger.Info("deviceMatch", "No Device match", AutoConfigPreferredDevice)
	}
	return ctrl.Result{}, nil
}

func (r *EdgeDeviceSignedRequestReconciler) markStatusPending(ctx context.Context, edsr *v1alpha1.EdgeDeviceSignedRequest) error {
	patch := client.MergeFrom(edsr.DeepCopy())
	now := v1.NewTime(time.Now())
	message := "device is waiting for approval"
	for i := range edsr.Status.Conditions {
		edsr.Status.Conditions[i].Status = "false"
	}
	edsr.Status.Conditions = append(edsr.Status.Conditions, v1alpha1.EdgeDeviceSignedRequestCondition{
		Type:               v1alpha1.EdgeDeviceSignedRequestStatusPending,
		Status:             "true",
		Message:            &message,
		LastTransitionTime: &now,
	})
	err := r.EdgedeviceSignedRequestRepository.PatchStatus(ctx, edsr, &patch)
	return err
}

func (r *EdgeDeviceSignedRequestReconciler) markStatusApproved(ctx context.Context, edsr *v1alpha1.EdgeDeviceSignedRequest) error {
	patch := client.MergeFrom(edsr.DeepCopy())
	now := v1.NewTime(time.Now())
	message := "device correctly approved"
	for i := range edsr.Status.Conditions {
		edsr.Status.Conditions[i].Status = "false"
	}
	edsr.Status.Conditions = append(edsr.Status.Conditions,
		v1alpha1.EdgeDeviceSignedRequestCondition{
			Type:               v1alpha1.EdgeDeviceSignedRequestStatusApproved,
			Status:             "true",
			Message:            &message,
			LastTransitionTime: &now,
		})
	err := r.EdgedeviceSignedRequestRepository.PatchStatus(ctx, edsr, &patch)
	return err
}

func IsPending(edsr *v1alpha1.EdgeDeviceSignedRequest) bool {
	if len(edsr.Status.Conditions) == 0 {
		return true
	}

	for _, status := range edsr.Status.Conditions {
		if status.Status == "True" && status.Type == v1alpha1.EdgeDeviceSignedRequestStatusPending {
			return true
		}
	}
	return false
}

// check if the device features are similar to preferred device properties
func (r *EdgeDeviceSignedRequestReconciler) compareDeviceFeaturesPreferred(registeringDeviceHWFeatures *v1alpha1.Features, prefferedDeviceHWFeatures *v1alpha1.EdgeDeviceProperties) bool {
	if registeringDeviceHWFeatures != nil && prefferedDeviceHWFeatures.Hardware != nil {
		if registeringDeviceHWFeatures.ModelName != "" && prefferedDeviceHWFeatures.OsModelName != "" {
			if registeringDeviceHWFeatures.ModelName == prefferedDeviceHWFeatures.OsModelName {
				return true
			}
		} else if registeringDeviceHWFeatures.Hardware.CPU.Architecture != "" && prefferedDeviceHWFeatures.Hardware.CPU.Architecture != "" {
			if registeringDeviceHWFeatures.Hardware.CPU.Architecture == prefferedDeviceHWFeatures.Hardware.CPU.Architecture {
				return true
			}
		} else if registeringDeviceHWFeatures.Hardware.CPU.ModelName != "" && prefferedDeviceHWFeatures.Hardware.CPU.ModelName != "" {
			if registeringDeviceHWFeatures.Hardware.CPU.ModelName == prefferedDeviceHWFeatures.Hardware.CPU.ModelName {
				return true
			}
		} else if registeringDeviceHWFeatures.Hardware.SystemVendor.Manufacturer != "" && prefferedDeviceHWFeatures.Hardware.SystemVendor.Manufacturer != "" {
			if registeringDeviceHWFeatures.Hardware.SystemVendor.Manufacturer == prefferedDeviceHWFeatures.Hardware.SystemVendor.Manufacturer {
				return true
			}
		} else if registeringDeviceHWFeatures.Hardware.SystemVendor.ProductName != "" && prefferedDeviceHWFeatures.Hardware.SystemVendor.ProductName != "" {
			if registeringDeviceHWFeatures.Hardware.SystemVendor.ProductName == prefferedDeviceHWFeatures.Hardware.SystemVendor.ProductName {
				return true
			}
		} else if registeringDeviceHWFeatures.Hardware.SystemVendor.SerialNumber != "" && prefferedDeviceHWFeatures.Hardware.SystemVendor.SerialNumber != "" {
			if registeringDeviceHWFeatures.Hardware.SystemVendor.SerialNumber == prefferedDeviceHWFeatures.Hardware.SystemVendor.SerialNumber {
				return true
			}
		}
	}
	return false
}

// patch AutoConfig
func (r *EdgeDeviceSignedRequestReconciler) patchEdgeAutoConfigStatus(ctx context.Context, autocfg *v1alpha1.EdgeAutoConfig, edgeDeviceName string, edgeDeviceState v1alpha1.EdgeDeviceState) error {
	patch := client.MergeFrom(autocfg.DeepCopy())

	new_device := v1alpha1.EdgeDevices{
		Name:            edgeDeviceName,
		EdgeDeviceState: edgeDeviceState,
	}
	autocfg.Status.EdgeDevices = append(autocfg.Status.EdgeDevices, new_device)

	err := r.EdgeAutoConfigRepository.PatchStatus(ctx, autocfg, &patch)
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeviceSignedRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDeviceSignedRequest{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}
