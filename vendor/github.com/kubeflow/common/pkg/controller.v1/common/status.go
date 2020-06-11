package common

import (
	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
)

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType) {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = make(map[apiv1.ReplicaType]*apiv1.ReplicaStatus)
	}

	jobStatus.ReplicaStatuses[rtype] = &apiv1.ReplicaStatus{}
}

// updateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
func updateJobReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		jobStatus.ReplicaStatuses[rtype].Active++
	case corev1.PodSucceeded:
		jobStatus.ReplicaStatuses[rtype].Succeeded++
	case corev1.PodFailed:
		jobStatus.ReplicaStatuses[rtype].Failed++
	}
}
