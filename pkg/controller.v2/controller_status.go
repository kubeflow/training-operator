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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
)

const (
	// tfJobCreatedReason is added in a tfjob when it is created.
	tfJobCreatedReason = "TFJobCreated"
	// tfJobSucceededReason is added in a tfjob when it is succeeded.
	tfJobSucceededReason = "TFJobSucceeded"
	// tfJobSucceededReason is added in a tfjob when it is running.
	tfJobRunningReason = "TFJobRunning"
	// tfJobSucceededReason is added in a tfjob when it is failed.
	tfJobFailedReason = "TFJobFailed"
)

// updateStatus updates the status of the tfjob.
func (tc *TFJobController) updateStatus(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, replicas int) error {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(tfjob.Status.TFReplicaStatuses[rtype].Succeeded)
	running := int(tfjob.Status.TFReplicaStatuses[rtype].Active)
	failed := int(tfjob.Status.TFReplicaStatuses[rtype].Failed)

	if rtype == tfv1alpha2.TFReplicaTypeWorker {
		// All workers are running, set StartTime.
		if running == replicas {
			now := metav1.Now()
			tfjob.Status.StartTime = &now
		}

		// Some workers are still running, leave a running condition.
		if running > 0 {
			msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}

		// All workers are succeeded, leave a succeeded condition.
		if expected == 0 {
			msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
			now := metav1.Now()
			tfjob.Status.CompletionTime = &now
			err := tc.updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}
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
	return nil
}

// updateTFJobStatus updates the status of the given TFJob.
func (tc *TFJobController) updateTFJobStatus(tfjob *tfv1alpha2.TFJob) error {
	_, err := tc.tfJobClientSet.KubeflowV1alpha2().TFJobs(tfjob.Namespace).Update(tfjob)
	return err
}

// updateTFJobConditions updates the conditions of the given tfjob.
func (tc *TFJobController) updateTFJobConditions(tfjob *tfv1alpha2.TFJob, conditionType tfv1alpha2.TFJobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&tfjob.Status, condition)
	return nil
}

// initializeTFReplicaStatuses initializes the TFReplicaStatuses for replica.
func initializeTFReplicaStatuses(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType) {
	if tfjob.Status.TFReplicaStatuses == nil {
		tfjob.Status.TFReplicaStatuses = make(map[tfv1alpha2.TFReplicaType]*tfv1alpha2.TFReplicaStatus)
	}

	tfjob.Status.TFReplicaStatuses[rtype] = &tfv1alpha2.TFReplicaStatus{}
}

// updateTFJobReplicaStatuses updates the TFJobReplicaStatuses according to the pod.
func updateTFJobReplicaStatuses(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, pod *v1.Pod) {
	switch pod.Status.Phase {
	case v1.PodRunning:
		tfjob.Status.TFReplicaStatuses[rtype].Active++
	case v1.PodSucceeded:
		tfjob.Status.TFReplicaStatuses[rtype].Succeeded++
	case v1.PodFailed:
		tfjob.Status.TFReplicaStatuses[rtype].Failed++
	}
}

// newCondition creates a new tfjob condition.
func newCondition(conditionType tfv1alpha2.TFJobConditionType, reason, message string) tfv1alpha2.TFJobCondition {
	return tfv1alpha2.TFJobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status tfv1alpha2.TFJobStatus, condType tfv1alpha2.TFJobConditionType) *tfv1alpha2.TFJobCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// setCondition updates the tfjob to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *tfv1alpha2.TFJobStatus, condition tfv1alpha2.TFJobCondition) {
	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// removeCondition removes the tfjob condition with the provided type.
func removementCondition(status *tfv1alpha2.TFJobStatus, condType tfv1alpha2.TFJobConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of tfjob conditions without conditions with the provided type.
func filterOutCondition(conditions []tfv1alpha2.TFJobCondition, condType tfv1alpha2.TFJobConditionType) []tfv1alpha2.TFJobCondition {
	var newConditions []tfv1alpha2.TFJobCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
