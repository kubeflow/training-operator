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

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	corev1 "k8s.io/api/core/v1"
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
)

func (tc *TFController) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	tfJobKey, err := KeyFunc(tfJob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for tfjob object %#v: %v", tfJob, err))
		return err
	}

	logger := commonutil.LoggerForJob(tfJob)

	worker0Completed, err := tc.IsWorker0Completed(tfJob, replicas)
	if err != nil {
		logger.Warnf("check if worker 0 completed error %v", err)
		return err
	}

	// Set StartTime.
	if jobStatus.StartTime == nil {
		now := metav1.Now()
		jobStatus.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if tfJob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
			logger.Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *tfJob.Spec.RunPolicy.ActiveDeadlineSeconds)
			tc.WorkQueue.AddAfter(tfJobKey, time.Duration(*tfJob.Spec.RunPolicy.ActiveDeadlineSeconds)*time.Second)
		}
	}
	// iterate the replica spec based on this order
	allTypes := []commonv1.ReplicaType{
		tfv1.TFReplicaTypeChief,
		tfv1.TFReplicaTypeEval,
		tfv1.TFReplicaTypeMaster,
		tfv1.TFReplicaTypePS,
		tfv1.TFReplicaTypeWorker,
	}
	for _, rtype := range allTypes {
		if replicas[rtype] == nil {
			continue
		}
		spec := replicas[rtype]
		status := jobStatus.ReplicaStatuses[rtype]

		// Expect to have `replicas - succeeded` pods alive.
		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logger.Infof("TFJob=%s/%s, ReplicaType=%s expected=%d, running=%d, failed=%d",
			tfJob.Namespace, tfJob.Name, rtype, expected, running, failed)

		// If the TFJob contains Chief or Master spec, then we will update the status
		// according to the Chief/Master spec.
		if ContainChieforMasterSpec(tfJob.Spec.TFReplicaSpecs) {
			if tfv1.IsChieforMaster(rtype) {
				if running > 0 {
					msg := fmt.Sprintf("TFJob %s/%s is running.",
						tfJob.Namespace, tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobRunning, tfJobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof(
							"Append tfjob condition error: %v", err)
						return err
					}
				}
				if expected == 0 {
					msg := fmt.Sprintf("TFJob %s/%s successfully completed.",
						tfJob.Namespace, tfJob.Name)
					tc.Recorder.Event(tfJob, corev1.EventTypeNormal, tfJobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobSucceeded, tfJobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					tfJobsSuccessCount.Inc()
				}
			}
		} else {
			if rtype == tfv1.TFReplicaTypeWorker {
				// Leave a succeeded condition for the following two cases:
				// 1. If default success policy is used and worker 0 has completed.
				// 2. If `SuccessPolicyAllWorkers` success policy is used and all workers are succeeded.
				if expected == 0 || (worker0Completed && *tfJob.Spec.SuccessPolicy != tfv1.SuccessPolicyAllWorkers) {
					msg := fmt.Sprintf("TFJob %s/%s successfully completed.",
						tfJob.Namespace, tfJob.Name)
					tc.Recorder.Event(tfJob, corev1.EventTypeNormal, tfJobSucceededReason, msg)
					if jobStatus.CompletionTime == nil {
						now := metav1.Now()
						jobStatus.CompletionTime = &now
					}
					err := commonutil.UpdateJobConditions(jobStatus,
						commonv1.JobSucceeded, tfJobSucceededReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
					tfJobsSuccessCount.Inc()
				} else if running > 0 {
					// Some workers are still running, leave a running condition.
					msg := fmt.Sprintf("TFJob %s/%s successfully completed.",
						tfJob.Namespace, tfJob.Name)
					err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, tfJobRunningReason, msg)
					if err != nil {
						commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
						return err
					}
				}
			}
		}

		if failed > 0 {
			restart := false
			for _, condition := range jobStatus.Conditions {
				if condition.Type == commonv1.JobRestarting {
					restart = true
				}
			}

			if restart {
				// job is restarting, no need to set it failed
				// we know it because we update the status condition when reconciling the replicas
				tfJobsFailureCount.Inc()
			} else {
				msg := fmt.Sprintf("TFJob %s/%s has failed because %d %s replica(s) failed.",
					tfJob.Namespace, tfJob.Name, failed, rtype)
				tc.Recorder.Event(tfJob, corev1.EventTypeNormal, tfJobFailedReason, msg)
				if tfJob.Status.CompletionTime == nil {
					now := metav1.Now()
					tfJob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus,
					commonv1.JobFailed, tfJobFailedReason, msg)
				if err != nil {
					commonutil.LoggerForJob(tfJob).Infof("Append tfjob condition error: %v", err)
					return err
				}
				tfJobsFailureCount.Inc()
			}
		}
	}
	// we assign the jobStatus to the tfJob.Status for testing purpose
	// it won't effect the main reconcile logic
	// because we already use oldStatus := jobStatus.DeepCopy() to record the oldStatus
	// and use !reflect.DeepEqual(*oldStatus, jobStatus) to decide whether to update the tfJob or not
	tfJob.Status = *jobStatus.DeepCopy()

	return nil
}

// UpdateJobStatusInApiServer updates the status of the given TFJob.
func (tc *TFController) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	tfJob, ok := job.(*tfv1.TFJob)
	if !ok {
		return fmt.Errorf("%v is not a type of TFJob", tfJob)
	}

	startTime := time.Now()
	logger := commonutil.LoggerForJob(tfJob)
	defer func() {
		logger.Infof("Finished updating TFJobs Status %q (%v)",
			tfJob.Name, time.Since(startTime))
	}()

	tfJob = tfJob.DeepCopy()
	tfJob.Status = *jobStatus.DeepCopy()

	_, err := tc.tfJobClientSet.KubeflowV1().TFJobs(tfJob.Namespace).UpdateStatus(tfJob)
	return err
}

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(jobStatus *commonv1.JobStatus, rtype commonv1.ReplicaType) {
	if jobStatus.ReplicaStatuses == nil {
		jobStatus.ReplicaStatuses = make(map[commonv1.ReplicaType]*commonv1.ReplicaStatus)
	}

	jobStatus.ReplicaStatuses[rtype] = &commonv1.ReplicaStatus{}
}

// updateJobReplicaStatuses updates the JobReplicaStatuses according to the pod.
func updateJobReplicaStatuses(jobStatus *commonv1.JobStatus, rtype commonv1.ReplicaType, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		jobStatus.ReplicaStatuses[rtype].Active++
	case corev1.PodSucceeded:
		jobStatus.ReplicaStatuses[rtype].Succeeded++
	case corev1.PodFailed:
		jobStatus.ReplicaStatuses[rtype].Failed++
	}
}

func isSucceeded(status commonv1.JobStatus) bool {
	return hasCondition(status, commonv1.JobSucceeded)
}

func isFailed(status commonv1.JobStatus) bool {
	return hasCondition(status, commonv1.JobFailed)
}

func hasCondition(status commonv1.JobStatus, condType commonv1.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
