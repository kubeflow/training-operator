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
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

// reconcilePods checks and updates pods for each given TFReplicaSpec.
// It will requeue the tfjob in case of an error while creating/deleting pods.
func (tc *TFJobController) reconcilePods(
	tfjob *tfv1alpha2.TFJob,
	pods []*v1.Pod,
	rtype tfv1alpha2.TFReplicaType,
	spec *tfv1alpha2.TFReplicaSpec) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}

	// Convert TFReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	// Get all pods for the type rt.
	pods = filterPodsForTFReplicaType(pods, rt)
	activePods := FilterActivePods(pods)
	succeeded, failed := getPodStatus(pods)
	runningPods := filterRunningPods(pods)

	// Expect to have `replicas - succeeded` pods alive.
	expected := *spec.Replicas - succeeded

	// All workers are succeeded, leave a succeeded condition.
	if expected == 0 && rtype == tfv1alpha2.TFReplicaTypeWorker {
		msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
		now := metav1.Now()
		tfjob.Status.CompletionTime = &now
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}

	// Some workers are still running, leave a running condition.
	if len(runningPods) > 0 && rtype == tfv1alpha2.TFReplicaTypeWorker {
		msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}

	// All workers are running, set StartTime
	if len(runningPods) == int(*spec.Replicas) && rtype == tfv1alpha2.TFReplicaTypeWorker {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
	}

	// Some workers or pss are failed , leave a failed condition.
	if failed > 0 {
		msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
		err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}

	// TODO(gaocegege): Use syncPods to sync all replicas to ensure that all replicas only has one pod running/succeeded.
	diff := len(activePods) - int(expected)

	if diff < 0 {
		// Need to create new pods.
		diffIndexes := getDiffPodIndexes(activePods, expected, loggerForTFJob(tfjob))
		if diff+len(diffIndexes) != 0 {
			// This should never happened.
			return fmt.Errorf("pods diff(%d) is not equal to length(%d) of diffIndexes", diff, len(diffIndexes))
		}

		expectationPodsKey := genExpectationPodsKey(tfjobKey, rt)
		err := tc.expectations.ExpectCreations(expectationPodsKey, diff)
		if err != nil {
			return err
		}

		for _, index := range diffIndexes {
			loggerForTFJob(tfjob).Infof("need to create new pod: %s-%s", rt, index)

			// Create OwnerReference.
			controllerRef := genOwnerReference(tfjob)

			// Append tfReplicaTypeLabel and tfReplicaIndexLabel labels.
			pTemplate := spec.Template.DeepCopy()

			labels := genLabels(tfjobKey)
			// Set type and index for the worker.
			labels[tfReplicaTypeLabel] = rt
			labels[tfReplicaIndexLabel] = index

			if pTemplate.Labels == nil {
				pTemplate.Labels = make(map[string]string)
			}

			for key, value := range labels {
				pTemplate.Labels[key] = value
			}

			// Generate TF_CONFIG JSON string.
			tfConfigStr := genTFConfigJSONStr(tfjob, rt, index)

			if tfConfigStr == "" {
				return nil
			}
			// Add TF_CONFIG environment variable.
			for i, _ := range pTemplate.Spec.Containers {
				if len(pTemplate.Spec.Containers[i].Env) == 0 {
					pTemplate.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
				}
				pTemplate.Spec.Containers[i].Env = append(pTemplate.Spec.Containers[i].Env, v1.EnvVar{
					Name:  "TF_CONFIG",
					Value: tfConfigStr,
				})
			}

			// Set restart policy
			if spec.RestartPolicy != tfv1alpha2.RestartPolicyExitCode {
				pTemplate.Spec.RestartPolicy = v1.RestartPolicy(spec.RestartPolicy)
			}

			err := tc.podControl.CreatePodsWithControllerRef(tfjob.Namespace, pTemplate, tfjob, controllerRef)
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
		}
	} else if diff > 0 {
		// TODO(CPH): Need to delete pods.
		loggerForTFJob(tfjob).Infof("need to delete pod but it is not implemented yet")
	}

	if tfjob.Status.TFReplicaStatuses == nil {
		tfjob.Status.TFReplicaStatuses = make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaStatus)
	}

	if _, ok := tfjob.Status.TFReplicaStatuses[rtype]; !ok {
		tfjob.Status.TFReplicaStatuses[rtype] = &tfv1alpha2.TFReplicaStatus{}
	}

	// Update the active status since we have created -diff pods during the loop.
	tfjob.Status.TFReplicaStatuses[rtype].Active = expected
	tfjob.Status.TFReplicaStatuses[rtype].Succeeded = succeeded
	tfjob.Status.TFReplicaStatuses[rtype].Failed = failed
	return nil
}

func (tc *TFJobController) syncPods(pods []*v1.Pod, replicas int, logger *log.Entry) {
	podSlices := getPodSlices(pods, replicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warning("We have to many pods for the worker %d", index)
			// Kill some
		}
		if len(podSlice) == 0 {
			// Create one
		}
		// We already have one, and check if it is succeede or something else.
		// pod := podSlice[0]
	}
}

func getPodSlices(pods []*v1.Pod, replicas int, logger *log.Entry) [][]*v1.Pod {
	podSlices := make([][]*v1.Pod, 0)
	for _, pod := range pods {
		if _, ok := pod.Labels[tfReplicaIndexLabel]; !ok {
			logger.Warning("The pod do not have the index label.")
		}
		index, err := strconv.Atoi(pod.Labels[tfReplicaIndexLabel])
		if err != nil {
			logger.Warning("Error when strconv.Atoi: %v", err)
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d", index)
		}

		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

// getDiffPodIndexes checks and gets diff indexes from desired and current.
func getDiffPodIndexes(activePods []*v1.Pod, replicas int32, logger *log.Entry) []string {
	desiredIndexes := make(map[string]string)

	for i := int32(0); i < replicas; i++ {
		desiredIndexes[fmt.Sprintf("%d", i)] = noHit
	}

	for _, pod := range activePods {
		if _, ok := pod.Labels[tfReplicaIndexLabel]; !ok {
			continue
		}

		index := pod.Labels[tfReplicaIndexLabel]
		indexNum, err := strconv.Atoi(index)
		if err != nil {
			logger.Warningf("The label index should be integer: %s", index)
		} else {
			// The situation should not happen.
			if indexNum < 0 || indexNum >= int(replicas) {
				logger.Warningf("The label index is not expected: %d", indexNum)
			}
		}

		if _, ok := desiredIndexes[index]; ok {
			desiredIndexes[index] = hit
		}
	}

	diffIndexes := []string{}
	for index, hit := range desiredIndexes {
		if hit == noHit {
			diffIndexes = append(diffIndexes, index)
		}
	}

	return diffIndexes
}

// getPodsForTFJob returns the set of pods that this tfjob should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned Pods are pointers into the cache.
func (tc *TFJobController) getPodsForTFJob(tfjob *tfv1alpha2.TFJob) ([]*v1.Pod, error) {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for tfjob object %#v: %v", tfjob, err))
		return nil, err
	}

	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: genLabels(tfjobKey),
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
	cm := NewPodControllerRefManager(tc.podControl, tfjob, selector, controllerKind, canAdoptFunc)
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
			loggerForTFJob(tfjob).Info("This pod's tfjob does not exists")
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
	// TODO(CPH): handle this gracefully.
}

// When a pod is deleted, enqueue the tfjob that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (tc *TFJobController) deletePod(obj interface{}) {
	// TODO(CPH): handle this gracefully.
}

// getPodStatus returns no of succeeded and failed pods running a job
func getPodStatus(pods []*v1.Pod) (succeeded, failed int32) {
	succeeded = int32(filterPods(pods, v1.PodSucceeded))
	failed = int32(filterPods(pods, v1.PodFailed))
	return
}

// filterPods returns pods based on their phase.
func filterPods(pods []*v1.Pod, phase v1.PodPhase) int {
	result := 0
	for i := range pods {
		if phase == pods[i].Status.Phase {
			result++
		}
	}
	return result
}
