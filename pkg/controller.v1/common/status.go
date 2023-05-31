package common

import (
	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/core"
	corev1 "k8s.io/api/core/v1"
)

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType) {
	core.InitializeReplicaStatuses(jobStatus, rtype)
}

// updateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
func updateJobReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType, pod *corev1.Pod) {
	core.UpdateJobReplicaStatuses(jobStatus, rtype, pod)
}
