/*
Copyright 2023 The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	apiv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	corev1 "k8s.io/api/core/v1"
)

// InitializeReplicaStatuses initializes the ReplicaStatuses for replica.
func InitializeReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType) {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = make(map[apiv1.ReplicaType]*apiv1.ReplicaStatus)
	}

	jobStatus.ReplicaStatuses[rtype] = &apiv1.ReplicaStatus{}
}

// UpdateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
func UpdateJobReplicaStatuses(jobStatus *apiv1.JobStatus, rtype apiv1.ReplicaType, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		if pod.DeletionTimestamp != nil {
			// when node is not ready, the pod will be in terminating state.
			// Count deleted Pods as failures to account for orphan Pods that
			// never have a chance to reach the Failed phase.
			jobStatus.ReplicaStatuses[rtype].Failed++
		} else {
			jobStatus.ReplicaStatuses[rtype].Active++
		}
	case corev1.PodSucceeded:
		jobStatus.ReplicaStatuses[rtype].Succeeded++
	case corev1.PodFailed:
		jobStatus.ReplicaStatuses[rtype].Failed++
	}
}
