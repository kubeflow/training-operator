package util

import commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

const (
	DefaultGangSchedulerName = "volcano"
)

func IsGangSchedulerSet(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, schedulerName string) bool {
	if len(schedulerName) == 0 {
		schedulerName = DefaultGangSchedulerName
	}

	for _, spec := range replicas {
		if spec.Template.Spec.SchedulerName != "" && spec.Template.Spec.SchedulerName == schedulerName {
			return true
		}
	}

	return false
}
