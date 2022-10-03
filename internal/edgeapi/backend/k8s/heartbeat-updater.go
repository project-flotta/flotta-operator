package k8s

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/go-multierror"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/hardware"
	"github.com/project-flotta/flotta-operator/models"
)

type Updater struct {
	repository RepositoryFacade
	recorder   record.EventRecorder
}

func (u *Updater) updateStatus(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, heartbeat *models.Heartbeat) error {
	var errors *multierror.Error
	edgeDeviceCopy := edgeDevice.DeepCopy()
	patch := client.MergeFrom(edgeDeviceCopy)

	playbookCopies := map[string]*v1alpha1.PlaybookExecution{}
	for _, heartbeatPlaybookExec := range heartbeat.PlaybookExecutions {
		pe, err := u.repository.GetPlaybookExecution(ctx, heartbeatPlaybookExec.Name, edgeDevice.Namespace) //TODO: how to get playbook exec namespace?
		if err != nil {
			multierror.Append(errors, fmt.Errorf("cannot find playbook execution with name %s: %v", heartbeatPlaybookExec.Name, err))
			continue
		}
		playbookCopies[heartbeatPlaybookExec.Name] = pe.DeepCopy()
	}

	edgeDevice.Status.LastSyncedResourceVersion = heartbeat.Version
	edgeDevice.Status.Phase = heartbeat.Status
	if heartbeat.Hardware != nil {
		edgeDevice.Status.Hardware = hardware.MapHardware(heartbeat.Hardware)
	}
	deployments := updateDeploymentStatuses(edgeDevice.Status.Workloads, heartbeat.Workloads)
	edgeDevice.Status.Workloads = deployments

	playbookExecutions := updatePlaybookExecutionStatuses(edgeDevice.Status.PlaybookExecutions, heartbeat.PlaybookExecutions)
	edgeDevice.Status.PlaybookExecutions = playbookExecutions

	edgeDevice.Status.UpgradeInformation = (*v1alpha1.UpgradeInformation)(heartbeat.Upgrade)
	if !reflect.DeepEqual(edgeDevice.Status, edgeDeviceCopy.Status) {
		return u.repository.PatchEdgeDeviceStatus(ctx, edgeDevice, &patch)
	}

	for _, heartbeatPlaybookExec := range heartbeat.PlaybookExecutions {
		peNew, err := u.repository.GetPlaybookExecution(ctx, heartbeatPlaybookExec.Name, edgeDevice.Namespace) //TODO: how to get playbook exec namespace?
		if err != nil {
			multierror.Append(errors, fmt.Errorf("cannot find playbook execution with name %s: %v", heartbeatPlaybookExec.Name, err))
		}

		if len(peNew.Status.Conditions) > 0 {
			peNew.Status.Conditions[len(peNew.Status.Conditions)-1].Status = v1.ConditionFalse
		}

		now := v1.Now()
		peCondition := v1alpha1.PlaybookExecutionCondition{
			Status:             v1.ConditionTrue,
			Type:               v1alpha1.PlaybookExecutionConditionType(heartbeatPlaybookExec.Status),
			LastTransitionTime: &now}

		peNew.Status.Conditions = append(peNew.Status.Conditions, peCondition)
		err = u.repository.PatchPlaybookExecution(ctx, playbookCopies[heartbeatPlaybookExec.Name], peNew) //TODO: how to get playbook exec namespace?
		if err != nil {
			multierror.Append(errors, fmt.Errorf("cannot patch playbook execution with name %s: %v", heartbeatPlaybookExec.Name, err))
			continue
		}
		peNewStatusType := v1alpha1.PlaybookExecutionConditionType(heartbeatPlaybookExec.Status)

		if peNewStatusType == v1alpha1.PlaybookExecutionSuccessfullyCompleted || peNewStatusType == v1alpha1.PlaybookExecutionCompletedWithError {
			//final state reached: remove label from dge device
			newEdgeDeviceLabels := edgeDevice.Labels
			delete(newEdgeDeviceLabels, "config/device-by-config")
			err = u.repository.UpdateEdgeDeviceLabels(ctx, edgeDevice, newEdgeDeviceLabels)
			multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

func (u *Updater) updateLabels(ctx context.Context, edgeDevice *v1alpha1.EdgeDevice, heartbeat *models.Heartbeat) error {
	return u.repository.UpdateEdgeDeviceLabels(ctx, edgeDevice, hardware.MapLabels(heartbeat.Hardware))
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

func updatePlaybookExecutionStatuses(oldPlaybookExecution []v1alpha1.PlaybookExec, playbookExecutions []*models.PlaybookExecutionStatus) []v1alpha1.PlaybookExec {
	playbookExecutionMap := make(map[string]v1alpha1.PlaybookExec)
	for _, peStatus := range oldPlaybookExecution {
		playbookExecutionMap[peStatus.Name] = peStatus
	}
	for _, status := range playbookExecutions {
		if newPlayExec, ok := playbookExecutionMap[status.Name]; ok {
			if string(newPlayExec.PlaybookExecutionStatus.Conditions[len(newPlayExec.PlaybookExecutionStatus.Conditions)-1].Type) != status.Status {
				newPlayExec.PlaybookExecutionStatus.Conditions[len(newPlayExec.PlaybookExecutionStatus.Conditions)-1].Status = v1.ConditionFalse
				now := v1.Now()
				newPlayExec.PlaybookExecutionStatus.Conditions = append(newPlayExec.PlaybookExecutionStatus.Conditions, v1alpha1.PlaybookExecutionCondition{Status: v1.ConditionTrue, Type: v1alpha1.PlaybookExecutionConditionType(status.Status), LastTransitionTime: &now})
			}
			playbookExecutionMap[status.Name] = newPlayExec
		}
	}
	var playbookExecs []v1alpha1.PlaybookExec //nolint
	for _, pe := range playbookExecutionMap {
		playbookExecs = append(playbookExecs, pe)
	}
	return playbookExecs
}
