package common

import (
	"testing"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestUpdateJobReplicaStatuses(t *testing.T) {
	jobStatus := apiv1.JobStatus{}
	initializeReplicaStatuses(&jobStatus, "worker")
	_, ok := jobStatus.ReplicaStatuses["worker"]
	// assert ReplicaStatus for "worker" exists
	assert.True(t, ok)
	setStatusForTest(&jobStatus, "worker", 2, 3, 1)
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Failed, int32(2))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Succeeded, int32(3))
	assert.Equal(t, jobStatus.ReplicaStatuses["worker"].Active, int32(1))
}

func setStatusForTest(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType, failed, succeeded, active int32) {
	pod := corev1.Pod{
		Status: corev1.PodStatus{},
	}
	var i int32
	for i = 0; i < failed; i++ {
		pod.Status.Phase = corev1.PodFailed
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
	for i = 0; i < succeeded; i++ {
		pod.Status.Phase = corev1.PodSucceeded
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
	for i = 0; i < active; i++ {
		pod.Status.Phase = corev1.PodRunning
		updateJobReplicaStatuses(jobStatus, rtype, &pod)
	}
}
