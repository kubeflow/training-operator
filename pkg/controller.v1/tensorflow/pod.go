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
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	train_util "github.com/kubeflow/tf-operator/pkg/util/train"
)

const (
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"

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

// reconcilePods checks and updates pods for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods.
func (tc *TFController) reconcilePods(
	tfjob *tfv1.TFJob,
	pods []*v1.Pod,
	rtype tfv1.TFReplicaType,
	spec *common.ReplicaSpec, rstatus map[string]v1.PodPhase) error {

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := tflogger.LoggerForReplica(tfjob, rt)
	// Get all pods for the type rt.
	pods, err := tc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	replicas := int(*spec.Replicas)
	restart := false
	worker0Completed := false
	masterRole := false

	initializeTFReplicaStatuses(tfjob, rtype)

	podSlices, podsToBeRemoved := tc.GetPodSlices(pods, replicas, logger)

	// Scale down
	if tfjob.Spec.EnableDynamicWorker && len(podsToBeRemoved) > 0 {
		// Currently only allow to scale down workers
		if rtype == tfv1.TFReplicaTypeWorker {
			logger.Infof("Removing %d workers", len(podsToBeRemoved))
			for _, pod := range podsToBeRemoved {
				err := tc.PodControl.DeletePod(tfjob.Namespace, pod.Name, tfjob)
				tc.Recorder.Eventf(tfjob, v1.EventTypeNormal, podScaleDown, "Pod: %v.%v is being removed", pod.Namespace, pod.Name)
				if err != nil {
					return err
				}
			}
		} else {
			logger.Warningf("Trying to scale down %s pods, which might be a mistake", rt)
		}
	}

	for index, podSlice := range podSlices {
		masterRole = false
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
			// TODO(gaocegege): Kill some pods.
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			// if master pod is present, select the master pod
			// if master is not present, first worker pod is selected as the master.
			if ContainChieforMasterSpec(tfjob) {
				if tfv1.IsChieforMaster(rtype) {
					masterRole = true
				}
			} else {
				if tfv1.IsWorker(rtype) && (index == 0) {
					masterRole = true
				}
			}
			err = tc.createNewPod(tfjob, rt, strconv.Itoa(index), spec, masterRole)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]
			// Get the exit code of the tensorflow container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == tfv1.DefaultContainerName && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					tc.Recorder.Eventf(tfjob, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == common.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := tc.PodControl.DeletePod(pod.Namespace, pod.Name, tfjob); err != nil {
						return err
					}
					restart = true
				}
			}

			// Check whether worker 0 is exited without error.
			if rtype == tfv1.TFReplicaTypeWorker && index == 0 &&
				exitCode == 0 && pod.Status.Phase == v1.PodSucceeded {
				worker0Completed = true
			}
			updateTFJobReplicaStatuses(tfjob, rtype, pod)
		}
	}

	return tc.updateStatusSingle(tfjob, rtype, replicas, restart, worker0Completed)
}

// createNewPod creates a new pod for the given index and type.
func (tc *TFController) createNewPod(tfjob *tfv1.TFJob, rt, index string, spec *common.ReplicaSpec, masterRole bool) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}
	expectationPodsKey := jobcontroller.GenExpectationPodsKey(tfjobKey, rt)
	err = tc.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}
	logger := tflogger.LoggerForReplica(tfjob, rt)
	// Create OwnerReference.
	controllerRef := tc.GenOwnerReference(tfjob)

	// Set type and index for the worker.
	labels := tc.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index

	if masterRole {
		labels[jobcontroller.JobRoleLabel] = "master"
	}

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = jobcontroller.GenGeneralName(tfjob.Name, rt, index)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := setClusterSpec(podTemplate, tfjob, rt, index); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		tc.Recorder.Event(tfjob, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	// if gang-scheduling is enabled:
	// 1. if user has specified other scheduler, we report a warning without overriding any fields.
	// 2. if no SchedulerName is set for pods, then we set the SchedulerName to "kube-batch".
	if tc.Config.EnableGangScheduling {
		if tc.isNonGangSchedulerSet(tfjob) {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			tc.Recorder.Event(tfjob, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		} else {
			podTemplate.Spec.SchedulerName = tc.Config.GangSchedulerName
		}

		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}
		podTemplate.Annotations[gangSchedulingPodGroupAnnotation] =
			jobcontroller.GenPodGroupName(tfjob.Name)
	}

	err = tc.PodControl.CreatePodsWithControllerRef(tfjob.Namespace, podTemplate, tfjob, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// setClusterSpec generates and sets TF_CONFIG for the given podTemplateSpec.
func setClusterSpec(podTemplateSpec *v1.PodTemplateSpec, tfjob *tfv1.TFJob, rt, index string) error {
	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(tfjob) {
		return nil
	}
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(tfjob, rt, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplateSpec.Spec.Containers {
		if podTemplateSpec.Spec.Containers[i].Name == tfv1.DefaultContainerName {
			if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
				podTemplateSpec.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
			}
			podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
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
	allTypes := []tfv1.TFReplicaType{
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

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *common.ReplicaSpec) {
	if spec.RestartPolicy == common.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}

func (tc *TFController) isNonGangSchedulerSet(tfjob *tfv1.TFJob) bool {
	for _, spec := range tfjob.Spec.TFReplicaSpecs {
		if spec.Template.Spec.SchedulerName != "" && spec.Template.Spec.SchedulerName != tc.Config.GangSchedulerName {
			return true
		}
	}
	return false
}
