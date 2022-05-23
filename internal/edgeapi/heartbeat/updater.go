package heartbeat

import (
	"context"
	"reflect"
	"time"

	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	mtrcs "github.com/project-flotta/flotta-operator/internal/common/metrics"
	"github.com/project-flotta/flotta-operator/internal/common/repository/edgedevice"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/hardware"
	"github.com/project-flotta/flotta-operator/models"
)

type Updater struct {
	deviceRepository edgedevice.Repository
	recorder         record.EventRecorder
	metrics          mtrcs.Metrics
}

func (u *Updater) updateStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, heartbeat *models.Heartbeat) error {
	u.metrics.RecordEdgeDevicePresence(edgeDevice.Namespace, edgeDevice.Name)

	edgeDeviceCopy := edgeDevice.DeepCopy()
	patch := client.MergeFrom(edgeDeviceCopy)

	edgeDevice.Status.LastSyncedResourceVersion = heartbeat.Version
	edgeDevice.Status.Phase = heartbeat.Status
	if heartbeat.Hardware != nil {
		edgeDevice.Status.Hardware = hardware.MapHardware(heartbeat.Hardware)
	}
	deployments := updateDeploymentStatuses(edgeDevice.Status.Workloads, heartbeat.Workloads)
	edgeDevice.Status.Workloads = deployments
	edgeDevice.Status.UpgradeInformation = (*v1alpha1.UpgradeInformation)(heartbeat.Upgrade)

	if !reflect.DeepEqual(edgeDevice.Status, edgeDeviceCopy.Status) {
		return u.deviceRepository.PatchStatus(ctx, edgeDevice, &patch)
	}
	return nil
}

func (u *Updater) updateLabels(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, heartbeat *models.Heartbeat) error {
	return u.deviceRepository.UpdateLabels(ctx, edgeDevice, hardware.MapLabels(heartbeat.Hardware))
}

func (u *Updater) processEvents(edgeDevice *v1alpha1.EdgeDevice, events []*models.EventInfo) {
	for _, event := range events {
		if event == nil {
			continue
		}

		if event.Type == models.EventInfoTypeWarn {
			u.recorder.Event(edgeDevice, v12.EventTypeWarning, event.Reason, event.Message)
		} else {
			u.recorder.Event(edgeDevice, v12.EventTypeNormal, event.Reason, event.Message)
		}
	}
}

func updateDeploymentStatuses(oldWorkloads []v1alpha1.Workload, workloads []*models.WorkloadStatus) []v1alpha1.Workload {
	edgeWorkloadMap := make(map[string]v1alpha1.Workload)
	for _, workloadStatus := range oldWorkloads {
		edgeWorkloadMap[workloadStatus.Name] = workloadStatus
	}
	for _, status := range workloads {
		if edgeWorkload, ok := edgeWorkloadMap[status.Name]; ok {
			if string(edgeWorkload.Phase) != status.Status {
				edgeWorkload.Phase = v1alpha1.EdgeWorkloadPhase(status.Status)
				edgeWorkload.LastTransitionTime = v1.Now()
			}
			edgeWorkload.LastDataUpload = v1.NewTime(time.Time(status.LastDataUpload))
			edgeWorkloadMap[status.Name] = edgeWorkload
		}
	}
	var edgeWorkloads []v1alpha1.Workload //nolint
	for _, edgeWorkload := range edgeWorkloadMap {
		edgeWorkloads = append(edgeWorkloads, edgeWorkload)
	}
	return edgeWorkloads
}
