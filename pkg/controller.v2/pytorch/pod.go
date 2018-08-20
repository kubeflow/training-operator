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

// Package controller provides a Kubernetes controller for a PyTorchJob resource.
package pytorch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/controller.v2/jobcontroller"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	train_util "github.com/kubeflow/tf-operator/pkg/util/train"
)

const (
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is setted in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
)

// reconcilePods checks and updates pods for each given PyTorchReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods.
func (pc *PyTorchController) reconcilePods(
	job *v1alpha2.PyTorchJob,
	pods []*v1.Pod,
	rtype v1alpha2.PyTorchReplicaType,
	spec *v1alpha2.PyTorchReplicaSpec, rstatus map[string]v1.PodPhase) error {

	// Convert PyTorchReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := pylogger.LoggerForReplica(job, rt)
	// Get all pods for the type rt.
	pods, err := pc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	replicas := int(*spec.Replicas)
	restart := false

	initializePyTorchReplicaStatuses(job, rtype)

	podSlices := getPodSlices(pods, replicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
			// TODO(gaocegege): Kill some pods.
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)
			err = pc.createNewPod(job, rtype, strconv.Itoa(index), spec)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]
			// Check if the pod is retryable.
			if spec.RestartPolicy == v1alpha2.RestartPolicyExitCode {
				var exitCode int32
				for _, status := range pod.Status.ContainerStatuses {
					state := status.State
					// Get the exit code of the pytorch container.
					if status.Name == v1alpha2.DefaultContainerName && state.Terminated != nil {
						exitCode = state.Terminated.ExitCode
					}
				}
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					logger.Infof("Need to restart the pod: %s-%d", rt, index)
					if err := pc.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
						return err
					}
					restart = true
				}
			}
			updatePyTorchJobReplicaStatuses(job, rtype, pod)
		}
	}

	return updateStatusSingle(job, rtype, replicas, restart)
}

// getPodSlices returns a slice, which element is the slice of pod.
func getPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, replicas)
	for _, pod := range pods {
		if _, ok := pod.Labels[replicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[replicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d", index)
		} else {
			podSlices[index] = append(podSlices[index], pod)
		}
	}
	return podSlices
}

// createNewPod creates a new pod for the given index and type.
func (pc *PyTorchController) createNewPod(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType, index string, spec *v1alpha2.PyTorchReplicaSpec) error {
	rt := strings.ToLower(string(rtype))
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for job object %#v: %v", job, err))
		return err
	}
	expectationPodsKey := jobcontroller.GenExpectationPodsKey(jobKey, rt)
	err = pc.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}
	logger := pylogger.LoggerForReplica(job, rt)
	// Create OwnerReference.
	controllerRef := pc.GenOwnerReference(job)

	// Set type and index for the worker.
	labels := pc.GenLabels(job.Name)
	labels[replicaTypeLabel] = rt
	labels[replicaIndexLabel] = index

	podTemplate := spec.Template.DeepCopy()
	totalReplicas := pc.GetTotalReplicas(job)
	// Set name for the template.
	podTemplate.Name = jobcontroller.GenGeneralName(job.Name, rt, index)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := setClusterSpec(podTemplate, job, totalReplicas, index, rtype); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		pc.Recorder.Event(job, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	err = pc.PodControl.CreatePodsWithControllerRef(job.Namespace, podTemplate, job, controllerRef)
	if err != nil && k8serrors.IsTimeout(err) {
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

func setClusterSpec(podTemplateSpec *v1.PodTemplateSpec, job *v1alpha2.PyTorchJob, totalReplicas int32, index string, rtype v1alpha2.PyTorchReplicaType) error {
	rank, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	masterPort, err := GetPortFromPyTorchJob(job, v1alpha2.PyTorchReplicaTypeMaster)
	if err != nil {
		return err
	}

	masterAddr := jobcontroller.GenGeneralName(job.Name, strings.ToLower(string(v1alpha2.PyTorchReplicaTypeMaster)), strconv.Itoa(0))
	if rtype == v1alpha2.PyTorchReplicaTypeMaster {
		if rank != 0 {
			return errors.New("Invalid config: There should be only a single master with index=0")
		}
		masterAddr = "localhost"
	} else {
		rank = rank + 1
	}

	for i := range podTemplateSpec.Spec.Containers {
		if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
			podTemplateSpec.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
		}
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  "MASTER_PORT",
			Value: strconv.Itoa(int(masterPort)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(int(totalReplicas)),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  "RANK",
			Value: strconv.Itoa(rank),
		})
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  "PYTHONUNBUFFERED",
			Value: "0",
		})
	}
	return nil
}

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *v1alpha2.PyTorchReplicaSpec) {
	if spec.RestartPolicy == v1alpha2.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}
