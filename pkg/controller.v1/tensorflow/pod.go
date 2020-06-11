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

// Package controller provides a Kubernetes controller for a TFJob resource.
package tensorflow

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	tflogger "github.com/kubeflow/common/pkg/util"
	train_util "github.com/kubeflow/common/pkg/util/train"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
)

const (
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"
	// gang scheduler name.

	gangSchedulerName                = "volcano"
	gangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"

	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
	// podScaleDown is the normal reason when scaling down number of pods
	podScaleDown = "PodScaleDown"
)

// SetClusterSpec generates and sets TF_CONFIG for the given podTemplateSpec.
func (tc *TFController) SetClusterSpec(job interface{}, podTemplate *v1.PodTemplateSpec, rtype, index string) error {
	tfjob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of MXJob", tfjob)
	}

	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(tfjob) {
		return nil
	}
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(tfjob, rtype, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplate.Spec.Containers {
		if podTemplate.Spec.Containers[i].Name == tfv1.DefaultContainerName {
			if len(podTemplate.Spec.Containers[i].Env) == 0 {
				podTemplate.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
			}
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, v1.EnvVar{
				Name:  tfConfig,
				Value: tfConfigStr,
			})
			break
		}
	}
	return nil
}

// isDistributed returns if the TFJob is a distributed training job.
// Ref https://github.com/kubeflow/tf-operator/issues/1078.
func isDistributed(tfjob *tfv1.TFJob) bool {
	replicas := tfjob.Spec.TFReplicaSpecs
	distributionCount := 0
	allTypes := []commonv1.ReplicaType{
		tfv1.TFReplicaTypeChief,
		tfv1.TFReplicaTypeEval,
		tfv1.TFReplicaTypeMaster,
		tfv1.TFReplicaTypePS,
		tfv1.TFReplicaTypeWorker,
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

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}

// IsWorker0Completed return true if pod of worker0 succeeded and exited with 0
func (tc *TFController) IsWorker0Completed(tfjob *tfv1.TFJob, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) (bool, error) {
	worker0Completed := false
	logger := tflogger.LoggerForReplica(tfjob, strings.ToLower(string(tfv1.TFReplicaTypeWorker)))

	pods, err := tc.GetPodsForJob(tfjob)
	if err != nil {
		tflogger.LoggerForJob(tfjob).Warnf("getPodsForTFJob error %v", err)
		return false, err
	}

	// Get all pods for the type rt.
	pods, err = tc.FilterPodsForReplicaType(pods, strings.ToLower(string(tfv1.TFReplicaTypeWorker)))
	if err != nil {
		return false, err
	}

	podSlices := tc.GetPodSlices(pods, int(*replicas[tfv1.TFReplicaTypeWorker].Replicas), logger)
	for index, podSlice := range podSlices {
		if len(podSlice) == 1 {
			pod := podSlice[0]
			// Get the exit code of the tensorflow container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == tfv1.DefaultContainerName && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
				}
			}

			if index == 0 && exitCode == 0 && pod.Status.Phase == v1.PodSucceeded {
				worker0Completed = true
			}
		}
	}
	return worker0Completed, nil
}

// PodRestart return true if pod failed with retryable exit code
func (tc *TFController) PodRestart(tfjob *tfv1.TFJob, spec *commonv1.ReplicaSpec, rtype commonv1.ReplicaType) (bool, error) {
	podRestart := false
	logger := tflogger.LoggerForReplica(tfjob, strings.ToLower(string(rtype)))

	pods, err := tc.GetPodsForJob(tfjob)
	if err != nil {
		tflogger.LoggerForJob(tfjob).Warnf("getPodsForTFJob error %v", err)
		return false, err
	}

	// Get all pods for the type rt.
	pods, err = tc.FilterPodsForReplicaType(pods, strings.ToLower(string(rtype)))
	if err != nil {
		return false, err
	}

	podSlices := tc.GetPodSlices(pods, int(*spec.Replicas), logger)
	for _, podSlice := range podSlices {
		if len(podSlice) == 1 {
			pod := podSlice[0]

			// Get the exit code of the tensorflow container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == tfv1.DefaultContainerName && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
				}
			}

			// Check if the pod is retryable.
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					podRestart = true
				}
			}
		}
	}
	return podRestart, nil

}
