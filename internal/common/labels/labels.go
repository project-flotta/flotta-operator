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
