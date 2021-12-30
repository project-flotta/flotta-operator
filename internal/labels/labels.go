package labels

import "strings"

const (
	DeviceNameLabel     = "devicename"
	DoesNotExistLabel   = "doesnotexist"
	workloadLabelPrefix = "workload/"
	selectorLabelPrefix = "selector/"
)

func WorkloadLabel(workloadName string) string {
	return workloadLabelPrefix + workloadName
}

func IsWorkloadLabel(label string) bool {
	return strings.HasPrefix(label, workloadLabelPrefix)
}

func IsSelectorLabel(label string) bool {
	return strings.HasPrefix(label, selectorLabelPrefix)
}

func CreateSelectorLabel(label string) string {
	return selectorLabelPrefix + label
}
