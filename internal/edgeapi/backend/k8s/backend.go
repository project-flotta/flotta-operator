package k8s

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/common/labels"
	backendapi "github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/hardware"
	"github.com/project-flotta/flotta-operator/models"
	"github.com/project-flotta/flotta-operator/pkg/mtls"
)

const (
	AuthzKey mtls.RequestAuthKey = "APIAuthzkey"
)

type backend struct {
	logger           *zap.SugaredLogger
	repository       RepositoryFacade
	assembler        *ConfigurationAssembler
	initialNamespace string
	heartbeatHandler *SynchronousHandler
}

func NewBackend(repository RepositoryFacade, assembler *ConfigurationAssembler,
	logger *zap.SugaredLogger, initialNamespace string, recorder record.EventRecorder) backendapi.EdgeDeviceBackend {
	return &backend{repository: repository,
		assembler:        assembler,
		logger:           logger,
		initialNamespace: initialNamespace,
		heartbeatHandler: NewSynchronousHandler(repository, recorder, logger)}
}

func (b *backend) GetRegistrationStatus(ctx context.Context, name, namespace string) (backendapi.RegistrationStatus, error) {
	edgeDevice, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return backendapi.Unregistered, nil
		}
		return backendapi.Unknown, err
	}

	if edgeDevice.DeletionTimestamp != nil {
		return backendapi.Unregistered, nil
	}

	return backendapi.Registered, nil
}

func (b *backend) GetConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error) {
	logger := b.logger.With("DeviceID", name, "Namespace", namespace)
	edgeDevice, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	return b.assembler.GetDeviceConfiguration(ctx, edgeDevice, logger)
}

func (b *backend) Enrol(ctx context.Context, name, namespace string, enrolmentInfo *models.EnrolmentInfo) (bool, error) {
	_, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err == nil {
		// Device is already created.
		return true, nil
	}

	edsr, err := b.repository.GetEdgeDeviceSignedRequest(ctx, name, b.initialNamespace)
	if err == nil {
		// Is already created, but not approved
		if edsr.Spec.TargetNamespace != namespace {
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
			TargetNamespace: namespace,
			Approved:        false,
			Features: &v1alpha1.Features{
				Hardware: hardware.MapHardware(enrolmentInfo.Features.Hardware),
			},
		},
	}

	return false, b.repository.CreateEdgeDeviceSignedRequest(ctx, edsr)
}

func (b *backend) GetTargetNamespace(ctx context.Context, name, identityNamespace string, matchesCertificate bool) (string, error) {
	logger := b.logger.With("DeviceID", name)
	namespace := identityNamespace
	if identityNamespace == b.initialNamespace && !matchesCertificate {
		// check if it's a valid device, shouldn't match
		esdr, err := b.repository.GetEdgeDeviceSignedRequest(ctx, name, b.initialNamespace)
		if err != nil {
			return "", err
		}
		if esdr.Spec.TargetNamespace != "" {
			namespace = esdr.Spec.TargetNamespace
		}
	}
	dvc, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", backendapi.NewNotApproved(err)
		}
		return "", err
	}

	if dvc == nil {
		return "", fmt.Errorf("device not found")
	}

	isInit := false
	if dvc.ObjectMeta.Labels[v1alpha1.EdgeDeviceSignedRequestLabelName] == v1alpha1.EdgeDeviceSignedRequestLabelValue {
		isInit = true
	}

	// the first time that tries to register should be able to use register certificate.
	if !isInit && !matchesCertificate {
		authKeyVal, _ := ctx.Value(AuthzKey).(mtls.RequestAuthVal)
		logger.With("certcn", authKeyVal.CommonName).Debug("Device tries to re-register with an invalid certificate")
		// At this moment, the registration certificate it's no longer valid,
		// because the CR is already created, and need to be a device
		// certificate.
		return "", fmt.Errorf("forbidden")
	}

	if isInit {
		logger.Info("EdgeDevice registered correctly for first time")
	} else {
		logger.Info("EdgeDevice renew registration correctly")
	}
	return namespace, nil
}

func (b *backend) Register(ctx context.Context, name, namespace string, registrationInfo *models.RegistrationInfo) error {
	logger := b.logger.With("DeviceID", name, "Namespace", namespace)
	dvc, err := b.repository.GetEdgeDevice(ctx, name, namespace)
	deviceCopy := dvc.DeepCopy()
	for key, val := range hardware.MapLabels(registrationInfo.Hardware) {
		deviceCopy.ObjectMeta.Labels[key] = val
	}
	delete(deviceCopy.Labels, v1alpha1.EdgeDeviceSignedRequestLabelName)

	err = b.repository.PatchEdgeDevice(ctx, dvc, deviceCopy)
	if err != nil {
		logger.With("err", err).Error("cannot update edgedevice")
		return err
	}

	err = b.updateDeviceStatus(ctx, dvc, func(device *v1alpha1.EdgeDevice) {
		device.Status.Hardware = hardware.MapHardware(registrationInfo.Hardware)
	})
	return err
}

func (b *backend) UpdateStatus(ctx context.Context, name, namespace string, heartbeat *models.Heartbeat) (bool, error) {
	return b.heartbeatHandler.Process(ctx, name, namespace, heartbeat)
}

func (b *backend) GetPlaybookExecutions(ctx context.Context, deviceID, namespace string) (*models.PlaybookExecutionsResponse, error) {
	logger := b.logger.With("DeviceID", deviceID, "Namespace", namespace)
	response := models.PlaybookExecutionsResponse{}
	edgeDevice, err := b.repository.GetEdgeDevice(ctx, deviceID, namespace)
	if err != nil {
		return nil, err
	}

	for labelName, labelValue := range edgeDevice.Labels {
		if labels.IsEdgeConfigLabel(labelName) {
			playbookExecution, err := b.repository.GetPlaybookExecution(ctx, labelValue, namespace)
			if err != nil {
				logger.Error(err, "cannot get playbook execution", "playbook execution name", labelValue, "namespace", namespace)
				return nil, err
			}
			playbookResponseItem := &models.PlaybookExecutionsResponseItems0{
				AnsiblePlaybook: string(playbookExecution.Spec.Playbook.Content),
			}
			response = append(response, playbookResponseItem)
		}
	}
	return &response, nil
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
