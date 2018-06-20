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
package controller

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/generator"
	train_util "github.com/kubeflow/tf-operator/pkg/util/train"
)

const (
	// tfConfig is the environment variable name of TensorFlow cluster spec.
	tfConfig = "TF_CONFIG"

	// podTemplateRestartPolicyReason is the warning reason when the restart
	// policy is setted in pod template.
	podTemplateRestartPolicyReason = "SettedPodTemplateRestartPolicy"
)

// reconcilePods checks and updates pods for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods.
func (tc *TFJobController) reconcilePods(
	tfjob *tfv1alpha2.TFJob,
	pods []*v1.Pod,
	rtype tfv1alpha2.TFReplicaType,
	spec *tfv1alpha2.TFReplicaSpec, rstatus map[string]v1.PodPhase) error {

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	// Get all pods for the type rt.
	pods = filterPodsForTFReplicaType(pods, rt)
	replicas := int(*spec.Replicas)
	restart := false

	initializeTFReplicaStatuses(tfjob, rtype)

	podSlices := getPodSlices(pods, replicas, loggerForReplica(tfjob, rt))
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			loggerForReplica(tfjob, rt).Warningf("We have too many pods for %s %d", rt, index)
			// TODO(gaocegege): Kill some pods.
		} else if len(podSlice) == 0 {
			loggerForReplica(tfjob, rt).Infof("Need to create new pod: %s-%d", rt, index)
			err := tc.createNewPod(tfjob, rt, strconv.Itoa(index), spec)
			if err != nil {
				return err
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]
			// Check if the pod is retryable.
			if spec.RestartPolicy == tfv1alpha2.RestartPolicyExitCode {
				var exitCode int32
				for _, status := range pod.Status.ContainerStatuses {
					state := status.State
					// Get the exit code of the tensorflow container.
					if status.Name == tfv1alpha2.DefaultContainerName && state.Terminated != nil {
						exitCode = state.Terminated.ExitCode
					}
				}
				if pod.Status.Phase == v1.PodFailed && train_util.IsRetryableExitCode(exitCode) {
					loggerForReplica(tfjob, rt).Infof("Need to restart the pod: %s-%d", rt, index)
					if err := tc.podControl.DeletePod(pod.Namespace, pod.Name, tfjob); err != nil {
						return err
					}
					restart = true
				}
			}
			updateTFJobReplicaStatuses(tfjob, rtype, pod)
		}
	}

	return updateStatusSingle(tfjob, rtype, replicas, restart)
}

// getPodSlices returns a slice, which element is the slice of pod.
func getPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, replicas)
	for _, pod := range pods {
		if _, ok := pod.Labels[tfReplicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(pod.Labels[tfReplicaIndexLabel])
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
func (tc *TFJobController) createNewPod(tfjob *tfv1alpha2.TFJob, rt, index string, spec *tfv1alpha2.TFReplicaSpec) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}
	expectationPodsKey := genExpectationPodsKey(tfjobKey, rt)
	err = tc.expectations.ExpectCreations(expectationPodsKey, 1)
	if err != nil {
		return err
	}

	// Create OwnerReference.
	controllerRef := generator.GenOwnerReference(tfjob)

	// Set type and index for the worker.
	labels := generator.GenLabels(tfjob.Name)
	labels[tfReplicaTypeLabel] = rt
	labels[tfReplicaIndexLabel] = index

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	podTemplate.Name = generator.GenGeneralName(tfjob.Name, rt, index)

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
		loggerForReplica(tfjob, rt).Warning(errMsg)
		tc.recorder.Event(tfjob, v1.EventTypeWarning, podTemplateRestartPolicyReason, errMsg)
	}
	setRestartPolicy(podTemplate, spec)

	err = tc.podControl.CreatePodsWithControllerRef(tfjob.Namespace, podTemplate, tfjob, controllerRef)
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

func setClusterSpec(podTemplateSpec *v1.PodTemplateSpec, tfjob *tfv1alpha2.TFJob, rt, index string) error {
	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := genTFConfigJSONStr(tfjob, rt, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}
	// Add TF_CONFIG environment variable.
	for i := range podTemplateSpec.Spec.Containers {
		if len(podTemplateSpec.Spec.Containers[i].Env) == 0 {
			podTemplateSpec.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
		}
		podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, v1.EnvVar{
			Name:  tfConfig,
			Value: tfConfigStr,
		})
	}
	return nil
}

func setRestartPolicy(podTemplateSpec *v1.PodTemplateSpec, spec *tfv1alpha2.TFReplicaSpec) {
	if spec.RestartPolicy == tfv1alpha2.RestartPolicyExitCode {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicyNever
	} else {
		podTemplateSpec.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
	}
}

// getPodsForTFJob returns the set of pods that this tfjob should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (tc *TFJobController) getPodsForTFJob(tfjob *tfv1alpha2.TFJob) ([]*v1.Pod, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: generator.GenLabels(tfjob.Name),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	pods, err := tc.podLister.Pods(tfjob.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := tc.tfJobClientSet.KubeflowV1alpha2().TFJobs(tfjob.Namespace).Get(tfjob.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != tfjob.UID {
			return nil, fmt.Errorf("original TFJob %v/%v is gone: got uid %v, wanted %v", tfjob.Namespace, tfjob.Name, fresh.UID, tfjob.UID)
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(tc.podControl, tfjob, selector, controllerKind, canAdoptFunc)
	return cm.ClaimPods(pods)
}

// filterPodsForTFReplicaType returns pods belong to a TFReplicaType.
func filterPodsForTFReplicaType(pods []*v1.Pod, tfReplicaType string) []*v1.Pod {
	var result []*v1.Pod

	tfReplicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	tfReplicaSelector.MatchLabels[tfReplicaTypeLabel] = tfReplicaType

	for _, pod := range pods {
		selector, _ := metav1.LabelSelectorAsSelector(tfReplicaSelector)
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result
}

func genExpectationPodsKey(tfjobKey, replicaType string) string {
	return tfjobKey + "/" + strings.ToLower(replicaType) + "/pods"
}

// RecheckDeletionTimestamp returns a CanAdopt() function to recheck deletion.
//
// The CanAdopt() function calls getObject() to fetch the latest value,
// and denies adoption attempts if that object has a non-nil DeletionTimestamp.
func RecheckDeletionTimestamp(getObject func() (metav1.Object, error)) func() error {
	return func() error {
		obj, err := getObject()
		if err != nil {
			return fmt.Errorf("can't recheck DeletionTimestamp: %v", err)
		}
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("%v/%v has just been deleted at %v", obj.GetNamespace(), obj.GetName(), obj.GetDeletionTimestamp())
		}
		return nil
	}
}

// When a pod is created, enqueue the tfjob that manages it and update its expectations.
func (tc *TFJobController) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.DeletionTimestamp != nil {
		// on a restart of the controller controller, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		// tc.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
		tfjob := tc.resolveControllerRef(pod.Namespace, controllerRef)
		if tfjob == nil {
			log.Info("This pod's tfjob does not exists")
			return
		}

		tfjobKey, err := KeyFunc(tfjob)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Failed to get the key of the tfjob: %v", err)
			return
		}

		if _, ok := pod.Labels[tfReplicaTypeLabel]; !ok {
			loggerForTFJob(tfjob).Info("This pod maybe not created by tf-operator")
			return
		}

		rtype := pod.Labels[tfReplicaTypeLabel]
		expectationPodsKey := genExpectationPodsKey(tfjobKey, rtype)

		tc.expectations.CreationObserved(expectationPodsKey)
		tc.enqueueTFJob(tfjob)

		return
	}

	// Otherwise, it's an orphan. Get a list of all matching controllers and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	// for _, tfjob := range tc.getPodJobs(pod) {
	// 	tc.enqueueTFJob(tfjob)
	// }
}

// When a pod is updated, figure out what tfjob/s manage it and wake them up.
// If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (tc *TFJobController) updatePod(old, cur interface{}) {
	curPod := cur.(*v1.Pod)
	oldPod := old.(*v1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if job := tc.resolveControllerRef(oldPod.Namespace, oldControllerRef); job != nil {
			log.Infof("pod ControllerRef updated: %v, %v", curPod, oldPod)
			tc.enqueueTFJob(job)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		job := tc.resolveControllerRef(curPod.Namespace, curControllerRef)
		if job == nil {
			return
		}
		log.Infof("pod has a ControllerRef: %v, %v", curPod, oldPod)
		tc.enqueueTFJob(job)
		return
	}
}

// When a pod is deleted, enqueue the tfjob that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (tc *TFJobController) deletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)

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
	tfJob := tc.resolveControllerRef(pod.Namespace, controllerRef)
	if tfJob == nil {
		return
	}
	tfJobKey, err := controller.KeyFunc(tfJob)
	if err != nil {
		return
	}

	if _, ok := pod.Labels[tfReplicaTypeLabel]; !ok {
		loggerForTFJob(tfJob).Info("This pod maybe not created by tf-operator")
		return
	}

	rtype := pod.Labels[tfReplicaTypeLabel]
	expectationPodsKey := genExpectationPodsKey(tfJobKey, rtype)

	tc.expectations.DeletionObserved(expectationPodsKey)
	tc.enqueueTFJob(tfJob)
}
