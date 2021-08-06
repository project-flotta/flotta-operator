package labels

func WorkloadLabel(workloadName string) string {
	return "workload/" + workloadName
}
