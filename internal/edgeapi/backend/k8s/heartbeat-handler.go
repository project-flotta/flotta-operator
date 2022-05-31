package k8s

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	backendapi "github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
)

type SynchronousHandler struct {
	repository RepositoryFacade
	updater    Updater
	logger     *zap.SugaredLogger
}

func NewSynchronousHandler(repository RepositoryFacade, recorder record.EventRecorder,
	logger *zap.SugaredLogger) *SynchronousHandler {
	return &SynchronousHandler{
		logger:     logger,
		repository: repository,
		updater: Updater{
			repository: repository,
			recorder:   recorder,
		},
	}
}

func (h *SynchronousHandler) Process(ctx context.Context, name, namespace string, notification backendapi.Notification) (bool, error) {
	logger := h.logger.With("DeviceID", name, "Namespace", namespace)
	hb := notification.Heartbeat
	logger.Debug("processing heartbeat", "content", hb, "retry", notification.Retry)
	edgeDevice, err := h.repository.GetEdgeDevice(ctx, name, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, err
		}
		return true, err
	}

	// Produce k8s events based on the device-worker events:
	if notification.Retry == 0 {
		h.updater.processEvents(edgeDevice, hb.Events)
	}

	err = h.updater.updateStatus(ctx, edgeDevice, hb)
	if err != nil {
		return true, err
	}
	err = h.updater.updateLabels(ctx, edgeDevice, hb)
	if err != nil {
		return true, err
	}

	return false, nil
}
