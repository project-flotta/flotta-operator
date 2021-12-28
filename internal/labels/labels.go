package labels

import "strings"

const workloadLabelPrefix = "workload/"

func WorkloadLabel(workloadName string) string {
	return workloadLabelPrefix + workloadName
}

func IsWorkloadLabel(label string) bool {
	return strings.HasPrefix(label, workloadLabelPrefix)
}
