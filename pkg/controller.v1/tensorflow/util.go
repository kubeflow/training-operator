// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tensorflow

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	trainingv1 "github.com/kubeflow/training-operator/pkg/apis/training/v1"

	corev1 "k8s.io/api/core/v1"
)

// GetPortFromTFJob gets the port of tensorflow container.
func GetPortFromTFJob(tfJob *trainingv1.TFJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := tfJob.Spec.TFReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == trainingv1.TFDefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == trainingv1.TFDefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return trainingv1.TFDefaultPort, nil
}

// ContainsChiefOrMasterSpec returns true if the tfjob contains chief or master spec.
func ContainsChiefOrMasterSpec(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) bool {
	if _, ok := replicas[trainingv1.TFReplicaTypeChief]; ok {
		return true
	} else if _, ok := replicas[trainingv1.TFReplicaTypeMaster]; ok {
		return true
	}
	return false
}

// originally from pkg/controller.v1/tensorflow/pod.go (deleted)
func getContainerExitCode(pod *corev1.Pod) int32 {
	var exitCode int32 = 0xbeef // magic number
	for _, status := range pod.Status.ContainerStatuses {
		state := status.State
		if status.Name == trainingv1.TFDefaultContainerName && state.Terminated != nil {
			exitCode = state.Terminated.ExitCode
		}
	}
	return exitCode
}

// originally from pkg/controller.v1/tensorflow/pod.go (deleted)
func setRestartPolicy(podTemplateSpec *corev1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}
}

// isDistributed returns if the TFJob is a distributed training job.
// Ref https://github.com/kubeflow/training-operator/issues/1078.
// originally from pkg/controller.v1/tensorflow/pod.go (deleted)
func isDistributed(tfjob *trainingv1.TFJob) bool {
	replicas := tfjob.Spec.TFReplicaSpecs
	distributionCount := 0
	allTypes := []commonv1.ReplicaType{
		trainingv1.TFReplicaTypeChief,
		trainingv1.TFReplicaTypeEval,
		trainingv1.TFReplicaTypeMaster,
		trainingv1.TFReplicaTypePS,
		trainingv1.TFReplicaTypeWorker,
	}
	// Check if there is only one replica.
	for _, typ := range allTypes {
		if replicas[typ] != nil {
			if replicas[typ].Replicas == nil {
				distributionCount++
			} else {
				distributionCount += int(*replicas[typ].Replicas)
			}
		}
	}
	return distributionCount != 1
}

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
// originally from pkg/controller.v1/tensorflow/status.go (deleted)
func initializeReplicaStatuses(jobStatus *commonv1.JobStatus, rtype commonv1.ReplicaType) {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = make(map[commonv1.ReplicaType]*commonv1.ReplicaStatus)
	}

	jobStatus.ReplicaStatuses[rtype] = &commonv1.ReplicaStatus{}
}

// updateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
// originally from pkg/controller.v1/tensorflow/status.go (deleted)
func updateJobReplicaStatuses(jobStatus *commonv1.JobStatus, rtype commonv1.ReplicaType, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		jobStatus.ReplicaStatuses[rtype].Active++
	case corev1.PodSucceeded:
		jobStatus.ReplicaStatuses[rtype].Succeeded++
	case corev1.PodFailed:
		jobStatus.ReplicaStatuses[rtype].Failed++
	}
}
