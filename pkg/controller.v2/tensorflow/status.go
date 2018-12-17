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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
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
	// tfJobRestarting is added in a tfjob when it is restarting.
	tfJobRestartingReason = "TFJobRestarting"
)

// updateStatus updates the status of the tfjob.
func updateStatusSingle(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, replicas int, restart, worker0Completed bool) error {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(tfjob.Status.TFReplicaStatuses[rtype].Succeeded)
	running := int(tfjob.Status.TFReplicaStatuses[rtype].Active)
	failed := int(tfjob.Status.TFReplicaStatuses[rtype].Failed)

	tflogger.LoggerForJob(tfjob).Infof("TFJob=%s, ReplicaType=%s expected=%d, running=%d, failed=%d",
		tfjob.Name, rtype, expected, running, failed)
	// All workers are running, set StartTime.
	if running == replicas && tfjob.Status.StartTime == nil {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
	}

	// If the TFJob contains Chief or Master spec, then we will update the status
	// according to the Chief/Master spec.
	if ContainChieforMasterSpec(tfjob) {
		if tfv1alpha2.IsChieforMaster(rtype) {
			if running > 0 {
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
			if expected == 0 {
				msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
				if tfjob.Status.CompletionTime == nil {
					now := metav1.Now()
					tfjob.Status.CompletionTime = &now
				}
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
		}
	} else {
		if rtype == tfv1alpha2.TFReplicaTypeWorker {
			// All workers are succeeded or worker 0 completed, leave a succeeded condition.
			if expected == 0 || worker0Completed {
				msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
				if tfjob.Status.CompletionTime == nil {
					now := metav1.Now()
					tfjob.Status.CompletionTime = &now
				}
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			} else if running > 0 {
				// Some workers are still running, leave a running condition.
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
		}
	}

	if failed > 0 {
		if restart {
			msg := fmt.Sprintf("TFJob %s is restarting.", tfjob.Name)
			err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRestarting, tfJobRestartingReason, msg)
			if err != nil {
				tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		} else {
			msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
			if tfjob.Status.CompletionTime == nil {
				now := metav1.Now()
				tfjob.Status.CompletionTime = &now
			}
			err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
			if err != nil {
				tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}
	}
	return nil
}

// updateTFJobStatus updates the status of the given TFJob.
func (tc *TFController) updateTFJobStatus(tfjob *tfv1alpha2.TFJob) error {
	_, err := tc.tfJobClientSet.KubeflowV1alpha2().TFJobs(tfjob.Namespace).Update(tfjob)
	return err
}

// updateTFJobConditions updates the conditions of the given tfjob.
func updateTFJobConditions(tfjob *tfv1alpha2.TFJob, conditionType tfv1alpha2.TFJobConditionType, reason, message string) error {
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
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

func hasCondition(status tfv1alpha2.TFJobStatus, condType tfv1alpha2.TFJobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isSucceeded(status tfv1alpha2.TFJobStatus) bool {
	return hasCondition(status, tfv1alpha2.TFJobSucceeded)
}

func isFailed(status tfv1alpha2.TFJobStatus) bool {
	return hasCondition(status, tfv1alpha2.TFJobFailed)
}

// setCondition updates the tfjob to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *tfv1alpha2.TFJobStatus, condition tfv1alpha2.TFJobCondition) {
	// Do nothing if TFJobStatus have failed condition
	if isFailed(*status) {
		return
	}

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

// filterOutCondition returns a new slice of tfjob conditions without conditions with the provided type.
func filterOutCondition(conditions []tfv1alpha2.TFJobCondition, condType tfv1alpha2.TFJobConditionType) []tfv1alpha2.TFJobCondition {
	var newConditions []tfv1alpha2.TFJobCondition
	for _, c := range conditions {
		if condType == tfv1alpha2.TFJobRestarting && c.Type == tfv1alpha2.TFJobRunning {
			continue
		}
		if condType == tfv1alpha2.TFJobRunning && c.Type == tfv1alpha2.TFJobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == tfv1alpha2.TFJobFailed || condType == tfv1alpha2.TFJobSucceeded) && c.Type == tfv1alpha2.TFJobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
