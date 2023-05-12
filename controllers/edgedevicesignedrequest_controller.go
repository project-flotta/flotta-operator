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
)

type EdgeDeviceSignedRequestReconciler struct {
	client.Client
	Scheme                            *runtime.Scheme
	EdgedeviceSignedRequestRepository edgedevicesignedrequest.Repository
	EdgeDeviceRepository              edgedevice.Repository
	EdgeAutoConfig                    edgeautoconfig.Repository
	MaxConcurrentReconciles           int
	AutoApproval                      bool
}

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
	autocfg, err := r.EdgeAutoConfig.ReadNS(ctx, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	autocfgcpy := autocfg.DeepCopy()

	AutoConfigConfigLabels := autocfgcpy.Spec.EdgeDeviceProperties
	DeviceLabels := map[string]string{
		v1alpha1.EdgeDeviceSignedRequestLabelName: v1alpha1.EdgeDeviceSignedRequestLabelValue,
	}

	for _, label := range DeviceLabels {
		if deviceLabelExistsInSlice(AutoConfigConfigLabels, label) {
			EdgeAutoConfigEdgedevices := autocfgcpy.Status.EdgeDevices
			if !deviceExistsInSlice(EdgeAutoConfigEdgedevices, edsr.Name) {
				logger.Info("checking edgeautoconfig CR for the new edgedevice")
				newDevice := managementv1alpha1.EdgeDevices{Name: edsr.Name}
				autocfgcpy.Status.EdgeDevices = append(autocfgcpy.Status.EdgeDevices, newDevice)
				err = r.EdgeAutoConfig.Patch(ctx, autocfg, autocfgcpy)
				if err != nil {
					logger.Error(err, "cannot patch edgeautoconfig status for the new edgedevice")
					return ctrl.Result{Requeue: true}, err
				}
			}
		}
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

// check if edsr device labels exists in array of preferred devices in CR
func deviceLabelExistsInSlice(arr []managementv1alpha1.EdgeDeviceProperties, val string) bool {
	for _, item := range arr {
		if item.Name == val {
			return true
		}
	}
	return false
}

// check if edsr.name or device id exists in EdgeAutoConfig CR
func deviceExistsInSlice(arr []managementv1alpha1.EdgeDevices, val string) bool {
	for _, item := range arr {
		if item.Name == val {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeDeviceSignedRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&managementv1alpha1.EdgeDeviceSignedRequest{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}
