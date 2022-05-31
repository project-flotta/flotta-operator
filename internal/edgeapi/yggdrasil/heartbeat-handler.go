package yggdrasil

import (
	"context"
	"time"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
)

//go:generate mockgen -package=yggdrasil -destination=mock_status-updater.go . StatusUpdater
type StatusUpdater interface {
	UpdateStatus(ctx context.Context, name, namespace string, notification backend.Notification) (bool, error)
}

type RetryingDelegatingHandler struct {
	delegate StatusUpdater
}

func NewRetryingDelegatingHandler(delegate StatusUpdater) *RetryingDelegatingHandler {
	return &RetryingDelegatingHandler{delegate: delegate}
}

func (h *RetryingDelegatingHandler) Process(ctx context.Context, name, namespace string, notification backend.Notification) error {
	// retry patching the edge device status
	var err error
	var retry bool
	for i := 1; i < 5; i++ {
		retry, err = h.delegate.UpdateStatus(ctx, name, namespace, notification)
		if err == nil {
			return nil
		}
		if !retry {
			break
		}

		notification.Retry++
		time.Sleep(time.Duration(i*50) * time.Millisecond)
	}
	return err
}
