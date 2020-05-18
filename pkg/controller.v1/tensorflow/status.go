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
	"time"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tflogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	// tfJobCreatedReason is added in a tfjob when it is created.
	tfJobCreatedReason = "TFJobCreated"
	// tfJobSucceededReason is added in a tfjob when it is succeeded.
	tfJobSucceededReason = "TFJobSucceeded"
	// tfJobRunningReason is added in a tfjob when it is running.
	tfJobRunningReason = "TFJobRunning"
	// tfJobFailedReason is added in a tfjob when it is failed.
	tfJobFailedReason = "TFJobFailed"
	// tfJobRestarting is added in a tfjob when it is restarting.
	tfJobRestartingReason = "TFJobRestarting"
)

var (
	tfJobsSuccessCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tf_operator_jobs_successful_total",
		Help: "Counts number of TF jobs successful",
	})
	tfJobsFailureCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tf_operator_jobs_failed_total",
		Help: "Counts number of TF jobs failed",
	})
	tfJobsRestartCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tf_operator_jobs_restarted_total",
		Help: "Counts number of TF jobs restarted",
	})
)

// updateStatus updates the status of the tfjob.
func (tc *TFController) updateStatusSingle(tfjob *tfv1.TFJob, rtype tfv1.TFReplicaType, replicas int, restart, worker0Completed bool) error {
	tfjobKey, err := KeyFunc(tfjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfjob, err))
		return err
	}

	commonType := common.ReplicaType(rtype)
	// Expect to have `replicas - succeeded` pods alive.
	expected := replicas - int(tfjob.Status.ReplicaStatuses[commonType].Succeeded)
	running := int(tfjob.Status.ReplicaStatuses[commonType].Active)
	failed := int(tfjob.Status.ReplicaStatuses[commonType].Failed)

	tflogger.LoggerForJob(tfjob).Infof("TFJob=%s, ReplicaType=%s expected=%d, running=%d, failed=%d",
		tfjob.Name, rtype, expected, running, failed)
	// set StartTime.
	if tfjob.Status.StartTime == nil {
		now := metav1.Now()
		tfjob.Status.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if tfjob.Spec.ActiveDeadlineSeconds != nil {
			tflogger.LoggerForJob(tfjob).Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *tfjob.Spec.ActiveDeadlineSeconds)
			tc.WorkQueue.AddAfter(tfjobKey, time.Duration(*tfjob.Spec.ActiveDeadlineSeconds)*time.Second)
		}
	}

	// If the TFJob contains Chief or Master spec, then we will update the status
	// according to the Chief/Master spec.
	if ContainChieforMasterSpec(tfjob) {
		if tfv1.IsChieforMaster(rtype) {
			if running > 0 {
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, common.JobRunning, tfJobRunningReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
			if expected == 0 {
				msg := fmt.Sprintf("TFJob %s successfully completed.", tfjob.Name)
				tc.Recorder.Event(tfjob, v1.EventTypeNormal, tfJobSucceededReason, msg)
				if tfjob.Status.CompletionTime == nil {
					now := metav1.Now()
					tfjob.Status.CompletionTime = &now
				}
				err := updateTFJobConditions(tfjob, common.JobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
				tfJobsSuccessCount.Inc()
			}
		}
	} else {
		if rtype == tfv1.TFReplicaTypeWorker {
			// All workers are succeeded or worker 0 completed, leave a succeeded condition.
			if expected == 0 || worker0Completed {
				msg := fmt.Sprintf("TFJob %s successfully completed.", tfjob.Name)
				tc.Recorder.Event(tfjob, v1.EventTypeNormal, tfJobSucceededReason, msg)
				if tfjob.Status.CompletionTime == nil {
					now := metav1.Now()
					tfjob.Status.CompletionTime = &now
				}
				err := updateTFJobConditions(tfjob, common.JobSucceeded, tfJobSucceededReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
				tfJobsSuccessCount.Inc()
			} else if running > 0 {
				// Some workers are still running, leave a running condition.
				msg := fmt.Sprintf("TFJob %s is running.", tfjob.Name)
				err := updateTFJobConditions(tfjob, common.JobRunning, tfJobRunningReason, msg)
				if err != nil {
					tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
					return err
				}
			}
		}
	}

	if failed > 0 {
		if restart {
			msg := fmt.Sprintf("TFJob %s is restarting because %d %s replica(s) failed.",
				tfjob.Name, failed, rtype)
			tc.Recorder.Event(tfjob, v1.EventTypeWarning, tfJobRestartingReason, msg)
			err := updateTFJobConditions(tfjob, common.JobRestarting, tfJobRestartingReason, msg)
			if err != nil {
				tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
			tfJobsFailureCount.Inc()
			tfJobsRestartCount.Inc()
		} else {
			msg := fmt.Sprintf("TFJob %s has failed because %d %s replica(s) failed.",
				tfjob.Name, failed, rtype)
			tc.Recorder.Event(tfjob, v1.EventTypeNormal, tfJobFailedReason, msg)
			if tfjob.Status.CompletionTime == nil {
				now := metav1.Now()
				tfjob.Status.CompletionTime = &now
			}
			err := updateTFJobConditions(tfjob, common.JobFailed, tfJobFailedReason, msg)
			if err != nil {
				tflogger.LoggerForJob(tfjob).Infof("Append tfjob condition error: %v", err)
				return err
			}
			tfJobsFailureCount.Inc()
		}
	}
	return nil
}

// updateTFJobStatus updates the status of the given TFJob.
func (tc *TFController) updateTFJobStatus(tfjob *tfv1.TFJob) error {
	startTime := time.Now()
	defer func() {
		tflogger.LoggerForJob(tfjob).Infof("Finished updating TFJobs Status %q (%v)",
			tfjob.Name, time.Since(startTime))
	}()
	_, err := tc.tfJobClientSet.KubeflowV1().TFJobs(tfjob.Namespace).UpdateStatus(tfjob)
	return err
}

// updateTFJobConditions updates the conditions of the given tfjob.
func updateTFJobConditions(tfjob *tfv1.TFJob, conditionType common.JobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&tfjob.Status, condition)
	return nil
}

// initializeTFReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeTFReplicaStatuses(tfjob *tfv1.TFJob, rtype tfv1.TFReplicaType) {
	commonType := common.ReplicaType(rtype)
	if tfjob.Status.ReplicaStatuses == nil {
		tfjob.Status.ReplicaStatuses = make(map[common.ReplicaType]*common.ReplicaStatus)
	}

	tfjob.Status.ReplicaStatuses[commonType] = &common.ReplicaStatus{}
}

// updateTFJobReplicaStatuses updates the TFJobReplicaStatuses according to the pod.
func updateTFJobReplicaStatuses(tfjob *tfv1.TFJob, rtype tfv1.TFReplicaType, pod *v1.Pod) {
	commonType := common.ReplicaType(rtype)
	switch pod.Status.Phase {
	case v1.PodRunning:
		tfjob.Status.ReplicaStatuses[commonType].Active++
	case v1.PodSucceeded:
		tfjob.Status.ReplicaStatuses[commonType].Succeeded++
	case v1.PodFailed:
		tfjob.Status.ReplicaStatuses[commonType].Failed++
	}
}

// newCondition creates a new tfjob condition.
func newCondition(conditionType common.JobConditionType, reason, message string) common.JobCondition {
	return common.JobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status common.JobStatus, condType common.JobConditionType) *common.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

func hasCondition(status common.JobStatus, condType common.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isSucceeded(status common.JobStatus) bool {
	return hasCondition(status, common.JobSucceeded)
}

func isFailed(status common.JobStatus) bool {
	return hasCondition(status, common.JobFailed)
}

// setCondition updates the tfjob to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *common.JobStatus, condition common.JobCondition) {
	// Do nothing if TFJobStatus is completed.
	if isFailed(*status) || isSucceeded(*status) {
		return
	}

	currentCond := getCondition(*status, condition.Type)

	if currentCond != nil {
		if currentCond.Status == condition.Status &&
			currentCond.Reason == condition.Reason &&
			currentCond.Message == condition.Message {
			// Do nothing if the condition does not change.
			return
		} else if currentCond.Status == condition.Status {
			// Do not update lastTransitionTime if the status of the condition doesn't change.
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}
	}

	// Append the updated condition to the status.Conditions.
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of tfjob conditions without conditions with the provided type.
func filterOutCondition(conditions []common.JobCondition, condType common.JobConditionType) []common.JobCondition {
	var newConditions []common.JobCondition
	for _, c := range conditions {
		if condType == common.JobRestarting && c.Type == common.JobRunning {
			continue
		}
		if condType == common.JobRunning && c.Type == common.JobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == common.JobFailed || condType == common.JobSucceeded) && c.Type == common.JobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
