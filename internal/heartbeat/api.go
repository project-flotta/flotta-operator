package heartbeat

import (
	"context"
	"github.com/jakub-dzon/k4e-operator/models"
)

type Notification struct {
	DeviceID  string
	Namespace string
	Heartbeat *models.Heartbeat
	Retry     int32
}

type Handler interface {
	Start()
	Process(ctx context.Context, notification Notification) error
}
