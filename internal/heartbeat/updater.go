package heartbeat

import (
	"context"
	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/internal/hardware"
	"github.com/jakub-dzon/k4e-operator/internal/repository/edgedevice"
	"github.com/jakub-dzon/k4e-operator/models"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Updater struct {
	deviceRepository edgedevice.Repository
	recorder         record.EventRecorder
}

func (u *Updater) updateStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, heartbeat *models.Heartbeat) error {
	patch := client.MergeFrom(edgeDevice.DeepCopy())

	edgeDevice.Status.LastSyncedResourceVersion = heartbeat.Version
	edgeDevice.Status.LastSeenTime = v1.NewTime(time.Time(heartbeat.Time))
	edgeDevice.Status.Phase = heartbeat.Status
	if heartbeat.Hardware != nil {
		edgeDevice.Status.Hardware = hardware.MapHardware(heartbeat.Hardware)
	}
	deployments := updateDeploymentStatuses(edgeDevice.Status.Deployments, heartbeat.Workloads)
	edgeDevice.Status.Deployments = deployments

	err := u.deviceRepository.PatchStatus(ctx, edgeDevice, &patch)
	return err
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

func updateDeploymentStatuses(oldDeployments []v1alpha1.Deployment, workloads []*models.WorkloadStatus) []v1alpha1.Deployment {
	deploymentMap := make(map[string]v1alpha1.Deployment)
	for _, deploymentStatus := range oldDeployments {
		deploymentMap[deploymentStatus.Name] = deploymentStatus
	}
	for _, status := range workloads {
		if deployment, ok := deploymentMap[status.Name]; ok {
			if string(deployment.Phase) != status.Status {
				deployment.Phase = v1alpha1.EdgeDeploymentPhase(status.Status)
				deployment.LastTransitionTime = v1.Now()
			}
			deployment.LastDataUpload = v1.NewTime(time.Time(status.LastDataUpload))
			deploymentMap[status.Name] = deployment
		}
	}
	var deployments []v1alpha1.Deployment
	for _, deployment := range deploymentMap {
		deployments = append(deployments, deployment)
	}
	return deployments
}
