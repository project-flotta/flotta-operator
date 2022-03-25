package informers

import (
	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	mtrcs "github.com/project-flotta/flotta-operator/internal/metrics"
)

type edgeDeviceEventHandler struct {
	metrics mtrcs.Metrics
}

func NewEdgeDeviceEventHandler(metrics mtrcs.Metrics) *edgeDeviceEventHandler {
	return &edgeDeviceEventHandler{metrics: metrics}
}

func (h *edgeDeviceEventHandler) OnAdd(obj interface{}) {
	edgeDevice := obj.(*v1alpha1.EdgeDevice)
	h.metrics.RegisterDeviceCounter(edgeDevice.Namespace, edgeDevice.Name)
}
func (h *edgeDeviceEventHandler) OnDelete(obj interface{}) {
	edgeDevice := obj.(*v1alpha1.EdgeDevice)
	h.metrics.RemoveDeviceCounter(edgeDevice.Namespace, edgeDevice.Name)
}

func (h *edgeDeviceEventHandler) OnUpdate(_, _ interface{}) {}
