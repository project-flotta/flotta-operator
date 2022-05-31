package yggdrasil

import (
	"context"
	"time"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
	"github.com/project-flotta/flotta-operator/models"
)

//go:generate mockgen -package=yggdrasil -destination=mock_status-updater.go . StatusUpdater
type StatusUpdater interface {
	UpdateStatus(ctx context.Context, name, namespace string, heartbeat *models.Heartbeat) (bool, error)
}

type RetryingDelegatingHandler struct {
	delegate StatusUpdater
}

func NewRetryingDelegatingHandler(delegate StatusUpdater) *RetryingDelegatingHandler {
	return &RetryingDelegatingHandler{delegate: delegate}
}

func (h *RetryingDelegatingHandler) Process(ctx context.Context, name, namespace string, heartbeat *models.Heartbeat) error {
	// retry patching the edge device status
	var err error
	var retry bool
	for i := 1; i < 5; i++ {
		childCtx := context.WithValue(ctx, backend.RetryContextKey, retry)
		retry, err = h.delegate.UpdateStatus(childCtx, name, namespace, heartbeat)
		if err == nil {
			return nil
		}
		if !retry {
			break
		}

		time.Sleep(time.Duration(i*50) * time.Millisecond)
	}
	return err
}
