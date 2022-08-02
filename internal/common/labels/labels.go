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
