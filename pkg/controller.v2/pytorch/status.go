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
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/pytorch/v1alpha2"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	// pytorchJobCreatedReason is added in a job when it is created.
	pytorchJobCreatedReason = "PyTorchJobCreated"
	// pytorchJobSucceededReason is added in a job when it is succeeded.
	pytorchJobSucceededReason = "PyTorchJobSucceeded"
	// pytorchJobSucceededReason is added in a job when it is running.
	pytorchJobRunningReason = "PyTorchJobRunning"
	// pytorchJobSucceededReason is added in a job when it is failed.
	pytorchJobFailedReason = "PyTorchJobFailed"
	// pytorchJobRestarting is added in a job when it is restarting.
	pytorchJobRestartingReason = "PyTorchJobRestarting"
)

// updateStatus updates the status of the job.
func updateStatusSingle(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType, replicas int, restart bool) error {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(job.Status.PyTorchReplicaStatuses[rtype].Succeeded)
	running := int(job.Status.PyTorchReplicaStatuses[rtype].Active)
	failed := int(job.Status.PyTorchReplicaStatuses[rtype].Failed)

	// All workers are running, set StartTime.
	if running == replicas && job.Status.StartTime == nil {
		now := metav1.Now()
		job.Status.StartTime = &now
	}

	if ContainMasterSpec(job) {
		if rtype == v1alpha2.PyTorchReplicaTypeMaster {
			if running > 0 {
				msg := fmt.Sprintf("PyTorchJob %s is running.", job.Name)
				err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobRunning, pytorchJobRunningReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
			}
			if expected == 0 {
				msg := fmt.Sprintf("PyTorchJob %s is successfully completed.", job.Name)
				now := metav1.Now()
				job.Status.CompletionTime = &now
				err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobSucceeded, pytorchJobSucceededReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	} else {
		if rtype == v1alpha2.PyTorchReplicaTypeWorker {
			// Some workers are still running, leave a running condition.
			if running > 0 {
				msg := fmt.Sprintf("PyTorchJob %s is running.", job.Name)
				err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobRunning, pytorchJobRunningReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
			}

			// All workers are succeeded, leave a succeeded condition.
			if expected == 0 {
				msg := fmt.Sprintf("PyTorchJob %s is successfully completed.", job.Name)
				now := metav1.Now()
				job.Status.CompletionTime = &now
				err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobSucceeded, pytorchJobSucceededReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	if failed > 0 {
		if restart {
			msg := fmt.Sprintf("PyTorchJob %s is restarting.", job.Name)
			err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobRestarting, pytorchJobRestartingReason, msg)
			if err != nil {
				pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
				return err
			}
		} else {
			msg := fmt.Sprintf("PyTorchJob %s is failed.", job.Name)
			err := updatePyTorchJobConditions(job, v1alpha2.PyTorchJobFailed, pytorchJobFailedReason, msg)
			if err != nil {
				pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
				return err
			}
		}
	}
	return nil
}

// updatePyTorchJobStatus updates the status of the given PyTorchJob.
func (pc *PyTorchController) updatePyTorchJobStatus(job *v1alpha2.PyTorchJob) error {
	_, err := pc.jobClientSet.Pytorch().PyTorchJobs(job.Namespace).Update(job)
	return err
}

// updatePyTorchJobConditions updates the conditions of the given job.
func updatePyTorchJobConditions(job *v1alpha2.PyTorchJob, conditionType v1alpha2.PyTorchJobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&job.Status, condition)
	return nil
}

// initializePyTorchReplicaStatuses initializes the PyTorchReplicaStatuses for replica.
func initializePyTorchReplicaStatuses(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType) {
	if job.Status.PyTorchReplicaStatuses == nil {
		job.Status.PyTorchReplicaStatuses = make(map[v1alpha2.PyTorchReplicaType]*v1alpha2.PyTorchReplicaStatus)
	}

	job.Status.PyTorchReplicaStatuses[rtype] = &v1alpha2.PyTorchReplicaStatus{}
}

// updatePyTorchJobReplicaStatuses updates the PyTorchJobReplicaStatuses according to the pod.
func updatePyTorchJobReplicaStatuses(job *v1alpha2.PyTorchJob, rtype v1alpha2.PyTorchReplicaType, pod *v1.Pod) {
	switch pod.Status.Phase {
	case v1.PodRunning:
		job.Status.PyTorchReplicaStatuses[rtype].Active++
	case v1.PodSucceeded:
		job.Status.PyTorchReplicaStatuses[rtype].Succeeded++
	case v1.PodFailed:
		job.Status.PyTorchReplicaStatuses[rtype].Failed++
	}
}

// newCondition creates a new job condition.
func newCondition(conditionType v1alpha2.PyTorchJobConditionType, reason, message string) v1alpha2.PyTorchJobCondition {
	return v1alpha2.PyTorchJobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status v1alpha2.PyTorchJobStatus, condType v1alpha2.PyTorchJobConditionType) *v1alpha2.PyTorchJobCondition {
	if len(status.Conditions) > 0 {
		return &status.Conditions[len(status.Conditions)-1]
	}
	return nil
}

func hasCondition(status v1alpha2.PyTorchJobStatus, condType v1alpha2.PyTorchJobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isSucceeded(status v1alpha2.PyTorchJobStatus) bool {
	return hasCondition(status, v1alpha2.PyTorchJobSucceeded)
}

func isFailed(status v1alpha2.PyTorchJobStatus) bool {
	return hasCondition(status, v1alpha2.PyTorchJobFailed)
}

// setCondition updates the job to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *v1alpha2.PyTorchJobStatus, condition v1alpha2.PyTorchJobCondition) {
	// Do nothing if PyTorchJobStatus have failed condition
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

// filterOutCondition returns a new slice of job conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha2.PyTorchJobCondition, condType v1alpha2.PyTorchJobConditionType) []v1alpha2.PyTorchJobCondition {
	var newConditions []v1alpha2.PyTorchJobCondition
	for _, c := range conditions {
		if condType == v1alpha2.PyTorchJobRestarting && c.Type == v1alpha2.PyTorchJobRunning {
			continue
		}
		if condType == v1alpha2.PyTorchJobRunning && c.Type == v1alpha2.PyTorchJobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == v1alpha2.PyTorchJobFailed || condType == v1alpha2.PyTorchJobSucceeded) && c.Type == v1alpha2.PyTorchJobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
