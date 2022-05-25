package backend

import (
	"context"
	"github.com/project-flotta/flotta-operator/internal/common/utils"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend/k8s"
	"github.com/project-flotta/flotta-operator/models"
	"go.uber.org/zap"
)

const (
	YggdrasilConnectionFinalizer = "yggdrasil-connection-finalizer"
	YggdrasilWorkloadFinalizer   = "yggdrasil-workload-finalizer"
)

type Backend interface {
	ShouldEdgeDeviceBeUnregistered(ctx context.Context, name, namespace string) (bool, error)
	GetDeviceConfiguration(ctx context.Context, name, namespace string) (*models.DeviceConfigurationMessage, error)
}

type backend struct {
	logger     *zap.SugaredLogger
	repository k8s.RepositoryFacade
	assembler  *ConfigurationAssembler
}

func NewBackend(repository k8s.RepositoryFacade, assembler *ConfigurationAssembler, logger *zap.SugaredLogger) Backend {
	return &backend{repository: repository, assembler: assembler, logger: logger}
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
