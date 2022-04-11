package labels

import (
	"strings"
)

const (
	DeviceNameLabel     = "devicename"
	DoesNotExistLabel   = "doesnotexist"
	WorkloadLabelPrefix = "workload/"
	SelectorLabelPrefix = "selector/"
)

func WorkloadLabel(workloadName string) string {
	return WorkloadLabelPrefix + workloadName
}

func IsWorkloadLabel(label string) bool {
	return strings.HasPrefix(label, WorkloadLabelPrefix)
}

func IsSelectorLabel(label string) bool {
	return strings.HasPrefix(label, SelectorLabelPrefix)
}

func CreateSelectorLabel(label string) string {
	return SelectorLabelPrefix + label
}

// GetPodmanLabels filter all the labels of the EdgeWorkload CR starting with prefix "podman/"
func GetPodmanLabels(workloadLabels map[string]string) map[string]string {
	labels := map[string]string{}
	for key, value := range workloadLabels {
		if strings.HasPrefix(key, "podman/") {
			labels[key[7:]] = value
		}
	}
	return labels
}
