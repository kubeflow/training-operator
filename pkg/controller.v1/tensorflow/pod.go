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

	"github.com/kubeflow/tf-operator/pkg/common/util"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	train_util "github.com/kubeflow/common/pkg/util/train"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// gang scheduler name.
	gangSchedulerName = "volcano"
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
	// gangSchedulingPodGroupAnnotation is the annotation key used by batch schedulers
	gangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"
)

var (
	tfJobsRestartCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tf_operator_jobs_restarted_total",
			Help: "Counts number of TF jobs restarted",
		},
		[]string{"job_namespace"},
	)
)

// reconcilePods checks and updates pods for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods.
func (tc *TFController) ReconcilePods(
	job interface{},
	jobStatus *commonv1.JobStatus,
	pods []*v1.Pod,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
) error {

	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := commonutil.LoggerForJob(tfJob)
	// Get all pods for the type rt.
	pods, err := tc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	masterRole := false
	//restart := false
	//worker0Completed := false

	initializeReplicaStatuses(jobStatus, rtype)

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := tc.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			// check if this replica is the master role
			masterRole = tc.IsMasterRole(replicas, rtype, index)
			// TODO: [should change to CreateNewPod]
			err = tc.createNewPod(tfJob, rt, strconv.Itoa(index), spec, masterRole, replicas)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = tc.PodControl.DeletePod(pod.Namespace, pod.Name, tfJob)
				if err != nil {
					return err
				}
			}
			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == tc.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					tc.Recorder.Eventf(tfJob, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := tc.PodControl.DeletePod(pod.Namespace, pod.Name, tfJob); err != nil {
						return err
					}

					// with common library framework, we have to handle restart status here
					// or we won't know which replica has been restarted in updateJobStatus after reconciling all replicas
					msg := fmt.Sprintf("TFJob %s is restarting because %s replica(s) failed.",
						tfJob.Name, rtype)
					tc.Recorder.Event(tfJob, corev1.EventTypeWarning, tfJobRestartingReason, msg)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, tfJobRestartingReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					tfJobsRestartCount.WithLabelValues(tfJob.Namespace).Inc()
				}
			}

			updateJobReplicaStatuses(jobStatus, rtype, pod)
		}
	}
	return nil
}

// createNewPod creates a new pod for the given index and type.
func (tc *TFController) createNewPod(tfjob *tfv1.TFJob, rt, index string, spec *commonv1.ReplicaSpec, masterRole bool,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}
	expectationPodsKey := expectation.GenExpectationPodsKey(tfjobKey, rt)
	err = tc.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}
	logger := commonutil.LoggerForReplica(tfjob, rt)
	// Create OwnerReference.
	controllerRef := tc.GenOwnerReference(tfjob)

	// Set type and index for the worker.
	labels := tc.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index

	if masterRole {
		labels[commonv1.JobRoleLabel] = "master"
	}

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = common.GenGeneralName(tfjob.Name, rt, index)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := tc.SetClusterSpec(tfjob, podTemplate, rt, index); err != nil {
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
		if util.IsGangSchedulerSet(replicas, gangSchedulerName) {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			tc.Recorder.Event(tfjob, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		} else {
			podTemplate.Spec.SchedulerName = gangSchedulerName
		}

		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}
		podTemplate.Annotations[gangSchedulingPodGroupAnnotation] = tfjob.GetName()
		podTemplate.Annotations[volcanoTaskSpecKey] = rt
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
		// Decrement the expected number of creates because the informer won't observe this pod
		logger.Infof(
			"Failed creation, decrementing expectations for tfjob %s/%s, key %s",
			tfjob.Namespace, tfjob.Name, expectationPodsKey)
		tc.Expectations.CreationObserved(expectationPodsKey)
		return err
	}
	return nil
}

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

func (tc *TFController) getPodSlices(tfjob *tfv1.TFJob, replicasNum *int32) ([][]*v1.Pod, error) {
	logger := commonutil.LoggerForReplica(tfjob, strings.ToLower(string(tfv1.TFReplicaTypeWorker)))

	pods, err := tc.Controller.GetPodsForJob(tfjob)
	if err != nil {
		commonutil.LoggerForJob(tfjob).Warnf("getPodsForTFJob error %v", err)
		return nil, err
	}

	// Get all pods for the type rt.
	pods, err = tc.FilterPodsForReplicaType(pods, strings.ToLower(string(tfv1.TFReplicaTypeWorker)))
	if err != nil {
		return nil, err
	}

	podSlices := tc.GetPodSlices(pods, int(*replicasNum), logger)
	return podSlices, nil
}

func getContainerExitCode(pod *v1.Pod) int32 {
	var exitCode int32 = 0xbeef // magic number
	for _, status := range pod.Status.ContainerStatuses {
		state := status.State
		if status.Name == tfv1.DefaultContainerName && state.Terminated != nil {
			exitCode = state.Terminated.ExitCode
		}
	}
	return exitCode
}

// IsWorker0Completed return true if pod of worker0 succeeded and exited with 0
func (tc *TFController) IsWorker0Completed(tfjob *tfv1.TFJob, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) (bool, error) {
	worker0Completed := false
	_, ok := replicas[tfv1.TFReplicaTypeWorker]
	if !ok {
		return true, nil
	}
	podSlices, err := tc.getPodSlices(tfjob, replicas[tfv1.TFReplicaTypeWorker].Replicas)
	if err != nil {
		return false, err
	}
	for index, podSlice := range podSlices {
		if len(podSlice) == 1 {
			pod := podSlice[0]
			exitCode := getContainerExitCode(pod)
			if index == 0 && exitCode == 0 && pod.Status.Phase == v1.PodSucceeded {
				worker0Completed = true
			}
		}
	}
	return worker0Completed, nil
}
