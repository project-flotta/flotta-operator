package k8s

import (
	"context"
	"fmt"
	backend2 "github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/utils"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/hardware"
	"github.com/project-flotta/flotta-operator/models"
	"github.com/project-flotta/flotta-operator/pkg/mtls"
)

const (
	YggdrasilConnectionFinalizer = "yggdrasil-connection-finalizer"
	YggdrasilWorkloadFinalizer   = "yggdrasil-workload-finalizer"

	AuthzKey mtls.RequestAuthKey = "APIAuthzkey"
)

type backend struct {
	logger           *zap.SugaredLogger
	repository       RepositoryFacade
	assembler        *ConfigurationAssembler
	initialNamespace string
}

func NewBackend(repository RepositoryFacade, assembler *ConfigurationAssembler, logger *zap.SugaredLogger, initialNamespace string) backend2.Backend {
	return &backend{repository: repository, assembler: assembler, logger: logger, initialNamespace: initialNamespace}
}

func (b *backend) ShouldEdgeDeviceBeUnregistered(ctx context.Context, name, namespace string) (bool, error) {
	edgeDevice, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		return false, err
	}

	if edgeDevice.DeletionTimestamp == nil || utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
		return false, nil
	}

	if utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilConnectionFinalizer) {
		err = b.repository.RemoveEdgeDeviceFinalizer(ctx, edgeDevice, YggdrasilConnectionFinalizer)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (b *backend) GetDeviceConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error) {
	logger := b.logger.With("DeviceID", name)
	edgeDevice, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	if edgeDevice.DeletionTimestamp != nil {
		if utils.HasFinalizer(&edgeDevice.ObjectMeta, YggdrasilWorkloadFinalizer) {
			err := b.repository.RemoveEdgeDeviceFinalizer(ctx, edgeDevice, YggdrasilWorkloadFinalizer)
			if err != nil {
				return nil, err
			}
		}
	}
	return b.assembler.GetDeviceConfiguration(ctx, edgeDevice, logger)
}

func (b *backend) EnrolEdgeDevice(ctx context.Context, name string, enrolmentInfo *models.EnrolmentInfo) (bool, error) {
	targetNamespace := b.initialNamespace
	if enrolmentInfo.TargetNamespace != nil {
		targetNamespace = *enrolmentInfo.TargetNamespace
	}
	_, err := b.repository.GetEdgeDevice(ctx, name, targetNamespace)
	if err == nil {
		// Device is already created.
		return true, nil
	}

	edsr, err := b.repository.GetEdgeDeviceSignedRequest(ctx, name, b.initialNamespace)
	if err == nil {
		// Is already created, but not approved
		if edsr.Spec.TargetNamespace != targetNamespace {
			_, err = b.repository.GetEdgeDevice(ctx, name, edsr.Spec.TargetNamespace)
			if err == nil {
				// Device is already created.
				return true, nil
			}
		}
		return false, nil
	}

	edsr = &v1alpha1.EdgeDeviceSignedRequest{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.initialNamespace,
		},
		Spec: v1alpha1.EdgeDeviceSignedRequestSpec{
			TargetNamespace: targetNamespace,
			Approved:        false,
			Features: &v1alpha1.Features{
				Hardware: hardware.MapHardware(enrolmentInfo.Features.Hardware),
			},
		},
	}

	return false, b.repository.CreateEdgeDeviceSignedRequest(ctx, edsr)
}

func (b *backend) InitializeEdgeDeviceRegistration(ctx context.Context, name, identityNamespace string, matchesCertificate bool) (bool, string, error) {
	logger := b.logger.With("DeviceID", name)
	namespace := identityNamespace
	if identityNamespace == b.initialNamespace && !matchesCertificate {
		// check if it's a valid device, shouldn't match
		esdr, err := b.repository.GetEdgeDeviceSignedRequest(ctx, name, b.initialNamespace)
		if err != nil {
			return false, "", err
		}
		if esdr.Spec.TargetNamespace != "" {
			namespace = esdr.Spec.TargetNamespace
		}
	}
	dvc, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, "", backend2.NewNotApproved(err)
		}
		return false, "", err
	}

	if dvc == nil {
		return false, "", fmt.Errorf("device not found")
	}

	isInit := false
	if dvc.ObjectMeta.Labels[v1alpha1.EdgeDeviceSignedRequestLabelName] == v1alpha1.EdgeDeviceSignedRequestLabelValue {
		isInit = true
	}

	// the first time that tries to register should be able to use register certificate.
	if !isInit && !matchesCertificate {
		authKeyVal, _ := ctx.Value(AuthzKey).(mtls.RequestAuthVal)
		logger.Debug("Device tries to re-register with an invalid certificate", "certcn", authKeyVal.CommonName)
		// At this moment, the registration certificate it's no longer valid,
		// because the CR is already created, and need to be a device
		// certificate.
		return false, "", fmt.Errorf("forbidden")
	}

	return isInit, namespace, nil
}

func (b *backend) FinalizeEdgeDeviceRegistration(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error {
	logger := b.logger.With("DeviceID", name)
	dvc, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	deviceCopy := dvc.DeepCopy()
	deviceCopy.Finalizers = []string{YggdrasilConnectionFinalizer, YggdrasilWorkloadFinalizer}
	for key, val := range hardware.MapLabels(registrationInfo.Hardware) {
		deviceCopy.ObjectMeta.Labels[key] = val
	}
	delete(deviceCopy.Labels, v1alpha1.EdgeDeviceSignedRequestLabelName)

	err = b.repository.PatchEdgeDevice(ctx, dvc, deviceCopy)
	if err != nil {
		logger.Error(err, "cannot update edgedevice")
		return err
	}

	err = b.updateDeviceStatus(ctx, dvc, func(device *v1alpha1.EdgeDevice) {
		device.Status.Hardware = hardware.MapHardware(registrationInfo.Hardware)
	})
	return err
}

func (b *backend) updateDeviceStatus(ctx context.Context, device *v1alpha1.EdgeDevice, updateFunc func(d *v1alpha1.EdgeDevice)) error {
	patch := client.MergeFrom(device.DeepCopy())
	updateFunc(device)
	err := b.repository.PatchEdgeDeviceStatus(ctx, device, &patch)
	if err == nil {
		return nil
	}

	// retry patching the edge device status
	for i := 1; i < 4; i++ {
		time.Sleep(time.Duration(i*50) * time.Millisecond)
		device2, err := b.repository.GetEdgeDevice(ctx, device.Name, device.Namespace)
		if err != nil {
			continue
		}
		patch = client.MergeFrom(device2.DeepCopy())
		updateFunc(device2)
		err = b.repository.PatchEdgeDeviceStatus(ctx, device2, &patch)
		if err == nil {
			return nil
		}
	}
	return err
}

