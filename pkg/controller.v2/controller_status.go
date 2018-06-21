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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfv1alpha2 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1alpha2"
	"github.com/kubeflow/tf-operator/pkg/generator"
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
	// workerCreatedReason is added in a status when it is  created.
	workerCreatedReason = "WorkerCreated"
	//workerSucceededReason is added in a status when it is  succeeded.
	workerSucceededReason = "WorkerSucceeded"
	//workerSucceededReason is added in a status when it is  running.
	workerRunningReason = "WorkerRunning"
	//workerFailedReason is added in a status when it is failed.
	workerFailedReason = "WorkerFailed"
	//workerPendingReason is added in a status when it is  pending.
	workerPendingReason = "WorkerPending"
	//psCreatedReason is added in a status when it is  created.
	psCreatedReason = "PSCreated"
	//psSucceededReason is added in a status when it is  succeeded.
	psSucceededReason = "PSSucceeded"
	//psRunningReason is added in a status when it is  running.
	psRunningReason = "PSRunning"
	//psFailedReason is added in a status when it is  failed.
	psFailedReason = "PSFailed"
	//psPendingReason is added in a status when it is  pending.
	psPendingReason = "PSPending"
	//chiefCreateReason is added in a status when it is created.
	chiefCreatedReason = "ChiefCreated"
	//chiefSucceededReason is added in a status when it is succeeded.
	chiefSucceededReason = "ChiefSucceeded"
	//chiefRunningReason is added in a status when it is running.
	chiefRunningReason = "ChiefRunning"
	//chiefFailedReason is added in a status when it is failed.
	chiefFailedReason = "ChiefFailed"
	//chiefPendingReason is added in a status when it is pending.
	chiefPendingReason = "ChiefPending"
)

// updateStatus updates the status of the tfjob.
func updateStatusSingle(tfjob *tfv1alpha2.TFJob, rtype tfv1alpha2.TFReplicaType, replicas int, restart bool) error {
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(tfjob.Status.TFReplicaStatuses[rtype].Succeeded)
	running := int(tfjob.Status.TFReplicaStatuses[rtype].Active)
	failed := int(tfjob.Status.TFReplicaStatuses[rtype].Failed)

	// All workers are running, set StartTime.
	if running == replicas && tfjob.Status.StartTime == nil {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
	}

	if generator.ContainChiefSpec(tfjob) {
		if rtype == tfv1alpha2.TFReplicaTypeChief {
			if running > 0 {
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
				if err != nil {
					loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
			if expected == 0 {
				msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
				now := metav1.Now()
				tfjob.Status.CompletionTime = &now
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
		}
	} else {
		if rtype == tfv1alpha2.TFReplicaTypeWorker {
			// Some workers are still running, leave a running condition.
			if running > 0 {
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
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
				err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
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
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		} else {
			msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
			err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
			if err != nil {
				loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
		}
	}
	return nil
}

//Update distributed training status
func updateStatusDistributed(tfjob *tfv1alpha2.TFJob, replicasStatus map[string]v1.PodPhase) error {
	if tfjob.Status.StartTime == nil {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
	}

	chiefReplicas, psReplicas, _ := getReplicasForTFJobType(tfjob)

	status := countTFJobTypeStatus(tfjob, replicasStatus)

	var realChiefRunningReason string
	var realChiefSucceededReason string
	var realChiefPendingReason string
	var realChiefFailedReason string

	//get real chief for TFJob
	if chiefReplicas == 0 {
		realChiefRunningReason = workerRunningReason
		realChiefFailedReason = workerFailedReason
		realChiefSucceededReason = workerSucceededReason
		realChiefPendingReason = workerPendingReason
	} else {
		realChiefRunningReason = chiefRunningReason
		realChiefFailedReason = chiefFailedReason
		realChiefSucceededReason = chiefSucceededReason
		realChiefPendingReason = chiefPendingReason
	}

	restartPolicy := getRestartPolicy(tfjob)
	if (status[psRunningReason] == psReplicas && status[realChiefRunningReason] == 1) ||
		(restartPolicy[tfv1alpha2.TFReplicaTypePS] != tfv1alpha2.RestartPolicyNever ||
			restartPolicy[tfv1alpha2.TFReplicaTypeChief] != tfv1alpha2.RestartPolicyNever) {
		//Running
		msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
		now := metav1.Now()
		tfjob.Status.CompletionTime = &now
		err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRunning, tfJobRunningReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}
	}
	// The chief is succeeded, thus we consider the TFJob is succeeded.
	if status[realChiefSucceededReason] == 1 {
		msg := fmt.Sprintf("TFJob %s is successfully completed.", tfjob.Name)
		now := metav1.Now()
		tfjob.Status.CompletionTime = &now
		err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobSucceeded, tfJobSucceededReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}

	}
	// PS or chief is failed and will not be restarted, thus we consider the TFJob is failed.
	if (restartPolicy[tfv1alpha2.TFReplicaTypeChief] == tfv1alpha2.RestartPolicyNever &&
		restartPolicy[tfv1alpha2.TFReplicaTypePS] == tfv1alpha2.RestartPolicyNever) &&
		(status[psFailedReason] != 0 || status[realChiefFailedReason] != 0) {
		msg := fmt.Sprintf("TFJob %s is failed.", tfjob.Name)
		now := metav1.Now()
		tfjob.Status.CompletionTime = &now
		err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobFailed, tfJobFailedReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}

	}
	// We do not set the status to restarting if the worker is failed, because the TFJob
	// could be running even if there is only one worker.
	if status[psRunningReason] == psReplicas && status[realChiefPendingReason] == 1 {
		//Restarting
		msg := fmt.Sprintf("TFJob %s is restarting ", tfjob.Name)
		now := metav1.Now()
		tfjob.Status.CompletionTime = &now
		err := updateTFJobConditions(tfjob, tfv1alpha2.TFJobRestarting, tfJobRestartingReason, msg)
		if err != nil {
			loggerForTFJob(tfjob).Infof("Append tfjob condition error: %v", err)
			return err
		}

	}

	return nil
}

//updateStatus updates the  tfjob status according to the replica status map.
func updateStatus(tfjob *tfv1alpha2.TFJob, rstatus map[string]v1.PodPhase) error {
	chiefReplicas, psReplicas, workerReplicas := getReplicasForTFJobType(tfjob)

	if psReplicas == 0 {
		if (chiefReplicas == 1 && workerReplicas == 0) || (chiefReplicas == 0 && workerReplicas == 1) {
			err := updateStatusSingle(tfjob, tfv1alpha2.TFReplicaTypeWorker, workerReplicas, false)
			if err != nil {
				return err
			}
		}
	} else {
		err := updateStatusDistributed(tfjob, rstatus)
		if err != nil {
			return err
		}
	}
	return nil
}

// get restartPolicy for tfjob
func getRestartPolicy(tfjob *tfv1alpha2.TFJob) map[tfv1alpha2.TFReplicaType]tfv1alpha2.RestartPolicy {
	restartPolicy := make(map[tfv1alpha2.TFReplicaType]tfv1alpha2.RestartPolicy)
	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		restartPolicy[rtype] = spec.RestartPolicy
	}

	return restartPolicy
}

//count status for TFJob(worker-0/chief/ps)
func countTFJobTypeStatus(tfjob *tfv1alpha2.TFJob, replicasStatus map[string]v1.PodPhase) map[string]int {
	status := make(map[string]int)

	_, psReplicas, _ := getReplicasForTFJobType(tfjob)

	initReplicasStatusesNum(status)

	chiefKey := string(tfv1alpha2.TFReplicaTypeChief)
	switch replicasStatus[chiefKey] {
	case v1.PodRunning:
		status[chiefRunningReason] = 1
	case v1.PodFailed:
		status[chiefFailedReason] = 1
	case v1.PodPending:
		status[chiefPendingReason] = 1
	case v1.PodSucceeded:
		status[chiefSucceededReason] = 1
	}

	workerKey := string(tfv1alpha2.TFReplicaTypeWorker) + "-0"
	switch replicasStatus[workerKey] {
	case v1.PodRunning:
		status[workerRunningReason] = 1
	case v1.PodSucceeded:
		status[workerSucceededReason] = 1
	case v1.PodPending:
		status[workerPendingReason] = 1
	case v1.PodFailed:
		status[workerFailedReason] = 1
	}
	for i := 0; i < psReplicas; i++ {
		psKey := string(tfv1alpha2.TFReplicaTypePS) + "-" + strconv.Itoa(i)
		switch replicasStatus[psKey] {
		case v1.PodRunning:
			status[psRunningReason]++
		case v1.PodFailed:
			status[psFailedReason]++
		case v1.PodPending:
			status[psPendingReason]++
		}
	}
	return status
}

//get Replicas for tfjob (chief/worker/ps)
func getReplicasForTFJobType(tfjob *tfv1alpha2.TFJob) (chiefReplicas, psReplicas, workerReplicas int) {
	for rtype, spec := range tfjob.Spec.TFReplicaSpecs {
		if rtype == tfv1alpha2.TFReplicaTypePS {
			psReplicas = int(*spec.Replicas)
		}
		if rtype == tfv1alpha2.TFReplicaTypeWorker {
			workerReplicas = int(*spec.Replicas)
		}
		if rtype == tfv1alpha2.TFReplicaTypeChief {
			chiefReplicas = int(*spec.Replicas)
		}
	}
	return
}

// initReplicasStatusesNum initializes the Chief/PS/Worker for status.
func initReplicasStatusesNum(status map[string]int) {
	status[workerPendingReason] = 0
	status[workerRunningReason] = 0
	status[workerFailedReason] = 0
	status[workerSucceededReason] = 0
	status[workerCreatedReason] = 0
	status[psCreatedReason] = 0
	status[psPendingReason] = 0
	status[psRunningReason] = 0
	status[psFailedReason] = 0
	status[psSucceededReason] = 0
	status[chiefFailedReason] = 0
	status[chiefCreatedReason] = 0
	status[chiefPendingReason] = 0
	status[chiefRunningReason] = 0
	status[chiefSucceededReason] = 0
}

// updateTFJobStatus updates the status of the given TFJob.
func (tc *TFJobController) updateTFJobStatus(tfjob *tfv1alpha2.TFJob) error {
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

func addTFJobReplicaStatuses(rtype tfv1alpha2.TFReplicaType, index int, pod *v1.Pod, rstatus map[string]v1.PodPhase) {
	var key string
	if index == -1 {
		key = string(rtype)
	} else {
		key = string(rtype) + "-" + strconv.Itoa(index)
	}
	rstatus[key] = pod.Status.Phase
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
	if len(status.Conditions) > 0 {
		return &status.Conditions[len(status.Conditions)-1]
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
		if condType == tfv1alpha2.TFJobRestarting && c.Type == tfv1alpha2.TFJobRunning {
			continue
		}
		if condType == tfv1alpha2.TFJobRunning && c.Type == tfv1alpha2.TFJobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
