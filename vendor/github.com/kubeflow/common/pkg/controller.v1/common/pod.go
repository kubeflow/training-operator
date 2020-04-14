// Copyright 2019 The Kubeflow Authors
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

package common

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"

	apiv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	trainutil "github.com/kubeflow/common/pkg/util/train"
)

const (
	// gang scheduler name.
	gangSchedulerName = "volcano"
	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is set in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
	// exitedWithCodeReason is the normal reason when the pod is exited because of the exit code.
	exitedWithCodeReason = "ExitedWithCode"
	// podTemplateSchedulerNameReason is the warning reason when other scheduler name is set
	// in pod templates with gang-scheduling enabled
	podTemplateSchedulerNameReason = "SettedPodTemplateSchedulerName"
)

// When a pod is created, enqueue the job that manages it and update its expectations.
func (jc *JobController) AddPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		// jc.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		job := jc.resolveControllerRef(pod.Namespace, controllerRef)

		logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

		if job == nil {
			if pod.Labels[apiv1.GroupNameLabel] == jc.Controller.GetGroupNameLabelValue() {
				logger.Info("This pod's job does not exist")
			}
			return
		}

		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			logger.Infof("Failed to get the jobkey: %v", err)
			return
		}

		if _, ok := pod.Labels[apiv1.ReplicaTypeLabel]; !ok {
			logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
			return
		}

		rtype := pod.Labels[apiv1.ReplicaTypeLabel]
		expectationPodsKey := GenExpectationPodsKey(jobKey, rtype)

		jc.Expectations.CreationObserved(expectationPodsKey)
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)

		return
	}

}

// When a pod is updated, figure out what job is managing it and wake it up.
// If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (jc *JobController) UpdatePod(old, cur interface{}) {
	curPod := cur.(*v1.Pod)
	oldPod := old.(*v1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	logger := commonutil.LoggerForPod(curPod, jc.Controller.GetAPIGroupVersionKind().Kind)
	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := jc.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			logger.Infof("pod ControllerRef updated: %v, %v", curPod, oldPod)
			jobKey, err := controller.KeyFunc(job)
			if err != nil {
				return
			}
			// TODO: we may need add backoff here
			jc.WorkQueue.Add(jobKey)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		job := jc.resolveControllerRef(curPod.Namespace, curControllerRef)
		if job == nil {
			return
		}
		logger.Debugf("pod has a ControllerRef: %v, %v", curPod, oldPod)
		jobKey, err := controller.KeyFunc(job)
		if err != nil {
			return
		}
		// TODO: we may need add backoff here
		jc.WorkQueue.Add(jobKey)
		return
	}
}

// When a pod is deleted, enqueue the job that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (jc *JobController) DeletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)

	logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new job will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a pod %+v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	job := jc.resolveControllerRef(pod.Namespace, controllerRef)
	if job == nil {
		return
	}
	jobKey, err := controller.KeyFunc(job)
	if err != nil {
		return
	}

	if _, ok := pod.Labels[apiv1.ReplicaTypeLabel]; !ok {
		logger.Infof("This pod maybe not created by %v", jc.Controller.ControllerName())
		return
	}

	rtype := pod.Labels[apiv1.ReplicaTypeLabel]
	expectationPodsKey := GenExpectationPodsKey(jobKey, rtype)

	jc.Expectations.DeletionObserved(expectationPodsKey)
	// TODO: we may need add backoff here
	jc.WorkQueue.Add(jobKey)
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (jc *JobController) FilterPodsForReplicaType(pods []*v1.Pod, replicaType string) ([]*v1.Pod, error) {
	var result []*v1.Pod

	replicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	replicaSelector.MatchLabels[apiv1.ReplicaTypeLabel] = replicaType

	for _, pod := range pods {
		selector, err := metav1.LabelSelectorAsSelector(replicaSelector)
		if err != nil {
			return nil, err
		}
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}

// getPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func (jc *JobController) GetPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, calculatePodSliceSize(pods, replicas))
	for _, pod := range pods {
		if _, ok := pod.Labels[apiv1.ReplicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, pod: %s/%s", index, pod.Namespace, pod.Name)
		}

		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

// calculatePodSliceSize compare max pod index with desired replicas and return larger size
func calculatePodSliceSize(pods []*v1.Pod, replicas int) int {
	size := 0
	for _, pod := range pods {
		if _, ok := pod.Labels[apiv1.ReplicaIndexLabel]; !ok {
			continue
		}
		index, err := strconv.Atoi(pod.Labels[apiv1.ReplicaIndexLabel])
		if err != nil {
			continue
		}
		size = MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return MaxInt(size+1, replicas)
}

// ReconcilePods checks and updates pods for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods.
func (jc *JobController) ReconcilePods(
	job interface{},
	jobStatus *apiv1.JobStatus,
	pods []*v1.Pod,
	rtype apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec,
	rstatus map[string]v1.PodPhase,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := commonutil.LoggerForReplica(metaObject, rt)
	// Get all pods for the type rt.
	pods, err := jc.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var masterRole bool

	initializeReplicaStatuses(jobStatus, rtype)

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := jc.GetPodSlices(pods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			logger.Infof("Need to create new pod: %s-%d", rt, index)

			// check if this replica is the master role
			masterRole = jc.Controller.IsMasterRole(replicas, rtype, index)
			err = jc.createNewPod(job, rt, strconv.Itoa(index), spec, masterRole, replicas)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = jc.Controller.DeletePod(job, pod)
				if err != nil {
					return err
				}
			}

			// Get the exit code of the container.
			var exitCode int32 = 0xbeef // magic number
			for _, status := range pod.Status.ContainerStatuses {
				state := status.State
				if status.Name == jc.Controller.GetDefaultContainerName() && state.Terminated != nil {
					exitCode = state.Terminated.ExitCode
					logger.Infof("Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
					jc.Recorder.Eventf(runtimeObject, v1.EventTypeNormal, exitedWithCodeReason, "Pod: %v.%v exited with code %v", pod.Namespace, pod.Name, exitCode)
				}
			}
			// Check if the pod is retryable.
			if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
				if pod.Status.Phase == v1.PodFailed && trainutil.IsRetryableExitCode(exitCode) {
					logger.Infof("Need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := jc.Controller.DeletePod(job, pod); err != nil {
						return err
					}
				}
			}

			updateJobReplicaStatuses(jobStatus, rtype, pod)
		}
	}
	return nil
}

// createNewPod creates a new pod for the given index and type.
func (jc *JobController) createNewPod(job interface{}, rt, index string, spec *apiv1.ReplicaSpec, masterRole bool,
	replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) error {

	metaObject, ok := job.(metav1.Object)
	if !ok {
		return fmt.Errorf("job is not a metav1.Object type")
	}
	runtimeObject, ok := job.(runtime.Object)
	if !ok {
		return fmt.Errorf("job is not a runtime.Object type")
	}
	jobKey, err := KeyFunc(metaObject)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	expectationPodsKey := GenExpectationPodsKey(jobKey, rt)
	err = jc.Expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}
	logger := commonutil.LoggerForReplica(metaObject, rt)

	// Set type and index for the worker.
	labels := jc.GenLabels(metaObject.GetName())
	labels[apiv1.ReplicaTypeLabel] = rt
	labels[apiv1.ReplicaIndexLabel] = index

	if masterRole {
		labels[apiv1.JobRoleLabel] = "master"
	}

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = GenGeneralName(metaObject.GetName(), rt, index)

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}

	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	if err := jc.Controller.SetClusterSpec(job, podTemplate, rt, index); err != nil {
		return err
	}

	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	if podTemplate.Spec.RestartPolicy != v1.RestartPolicy("") {
		errMsg := "Restart policy in pod template will be overwritten by restart policy in replica spec"
		logger.Warning(errMsg)
		jc.Recorder.Event(runtimeObject, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	// if gang-scheduling is enabled:
	// 1. if user has specified other scheduler, we report a warning without overriding any fields.
	// 2. if no SchedulerName is set for pods, then we set the SchedulerName to "volcano".
	if jc.Config.EnableGangScheduling {
		if isNonGangSchedulerSet(replicas) {
			errMsg := "Another scheduler is specified when gang-scheduling is enabled and it will not be overwritten"
			logger.Warning(errMsg)
			jc.Recorder.Event(runtimeObject, v1.EventTypeWarning, podTemplateSchedulerNameReason, errMsg)
		} else {
			podTemplate.Spec.SchedulerName = gangSchedulerName
		}
	}
	controllerRef := jc.GenOwnerReference(metaObject)

	err = jc.createPodWithControllerRef(metaObject.GetNamespace(), podTemplate, runtimeObject, controllerRef)
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

func (jc *JobController) createPodWithControllerRef(namespace string, template *v1.PodTemplateSpec,
	controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return jc.createPod("", namespace, template, controllerObject, controllerRef)
}

func (jc *JobController) createPod(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	pod, err := GetPodFromTemplate(template, object, controllerRef)
	pod.Namespace = namespace
	if err != nil {
		return err
	}
	if len(nodeName) != 0 {
		pod.Spec.NodeName = nodeName
	}
	if labels.Set(pod.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create pods, no labels")
	}
	if err := jc.Controller.CreatePod(object, pod); err != nil {
		jc.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreatePodReason, "Error creating: %v", err)
		return err
	} else {
		logger := commonutil.LoggerForPod(pod, jc.Controller.GetAPIGroupVersionKind().Kind)
		accessor, err := meta.Accessor(object)
		if err != nil {
			logger.Errorf("parentObject does not have ObjectMeta, %v", err)
			return nil
		}
		logger.Infof("Controller %v created pod %v", accessor.GetName(), pod.Name)
		jc.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreatePodReason, "Created pod: %v", pod.Name)
	}
	return nil
}

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *apiv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	if spec.RestartPolicy == apiv1.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}

func isNonGangSchedulerSet(replicas map[apiv1.ReplicaType]*apiv1.ReplicaSpec) bool {
	for _, spec := range replicas {
		if spec.Template.Spec.SchedulerName != "" && spec.Template.Spec.SchedulerName != gangSchedulerName {
			return true
		}
	}
	return false
}
