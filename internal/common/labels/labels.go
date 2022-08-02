package labels

import (
	"strings"
)

const (
	DeviceNameLabel       = "devicename"
	DoesNotExistLabel     = "doesnotexist"
	WorkloadLabelPrefix   = "workload/"
	SelectorLabelPrefix   = "selector/"
	EdgeConfigLabelPrefix = "edgeconfig/"
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

func IsEdgeConfigLabel(label string) bool {
	return strings.HasPrefix(label, EdgeConfigLabelPrefix)
}

// GetEdgeConfigLabels filter all the labels of the EdgeDevice CR starting with prefix "edgeconfig/"
func GetEdgeConfigLabels(workloadLabels map[string]string) map[string]string {
	labels := map[string]string{}
	for key, value := range workloadLabels {
		if strings.HasPrefix(key, EdgeConfigLabelPrefix) {
			labels[key[len(EdgeConfigLabelPrefix):]] = value
		}
	}
	return labels
}
