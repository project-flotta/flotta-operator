package indexer

import (
	"strings"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/labels"
	flottalabels "github.com/project-flotta/flotta-operator/internal/labels"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DeploymentByDeviceIndexKey is the name of the indexer for deployments by device
	DeploymentByDeviceIndexKey = "deployment-by-device"

	// DeviceByWorkloadIndexKey is the key of the indexer for devices by workload
	DeviceByWorkloadIndexKey = "device-by-workload"
)

func DeploymentByDeviceIndexFunc(obj ctrlruntimeclient.Object) []string {
	deployment := obj.(*v1alpha1.EdgeDeployment)
	var keys []string
	for name, value := range deployment.Labels {
		if flottalabels.IsSelectorLabel(name) {
			keys = append(keys, CreateDeploymentIndexKey(name, value))
		}
	}
	return keys
}

func DeviceByWorkloadIndexFunc(obj ctrlruntimeclient.Object) []string {
	device := obj.(*v1alpha1.EdgeDevice)
	var keys []string
	// iterate over labels map and return a list of workloads as keys
	for name := range device.Labels {
		if flottalabels.IsWorkloadLabel(name) {
			keys = append(keys, CreateDeviceIndexKey(name))
		}
	}
	return keys
}

// CreateDeviceIndexKey creates a key for the device index which is basically the workload name
func CreateDeviceIndexKey(label string) string {
	return strings.TrimPrefix(label, labels.WorkloadLabelPrefix)
}

// CreateDeploymentIndexKey creates a key for the deployment index
// The key is of the form: device or label
func CreateDeploymentIndexKey(label, value string) string {
	suffix := strings.TrimPrefix(label, labels.SelectorLabelPrefix)
	if suffix == labels.DeviceNameLabel {
		return value
	}
	return suffix
}
