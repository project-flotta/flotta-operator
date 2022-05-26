package yggdrasil

import (
	"context"
	"time"

	"github.com/project-flotta/flotta-operator/internal/edgeapi/backend"
)

type RetryingDelegatingHandler struct {
	delegate backend.HeartbeatHandler
}

func NewRetryingDelegatingHandler(delegate backend.HeartbeatHandler) *RetryingDelegatingHandler {
	return &RetryingDelegatingHandler{delegate: delegate}
}

func (h *RetryingDelegatingHandler) Process(ctx context.Context, notification backend.Notification) error {
	// retry patching the edge device status
	var err error
	var retry bool
	for i := 1; i < 5; i++ {
		retry, err = h.delegate.Process(ctx, notification)
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
