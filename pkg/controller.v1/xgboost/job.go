/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xgboost

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	commonutil "github.com/kubeflow/common/pkg/util"
	logger "github.com/kubeflow/common/pkg/util"
	xgboostv1 "github.com/kubeflow/tf-operator/pkg/apis/xgboost/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reasons for job events.
const (
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"
	// xgboostJobCreatedReason is added in a job when it is created.
	xgboostJobCreatedReason = "XGBoostJobCreated"

	xgboostJobSucceededReason  = "XGBoostJobSucceeded"
	xgboostJobRunningReason    = "XGBoostJobRunning"
	xgboostJobFailedReason     = "XGBoostJobFailed"
	xgboostJobRestartingReason = "XGBoostJobRestarting"
)

// DeleteJob deletes the job
func (r *XGBoostJobReconciler) DeleteJob(job interface{}) error {
	xgboostjob, ok := job.(*xgboostv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}
	if err := r.Delete(context.Background(), xgboostjob); err != nil {
		r.recorder.Eventf(xgboostjob, corev1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		r.Log.Error(err, "failed to delete job", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
		return err
	}
	r.recorder.Eventf(xgboostjob, corev1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", xgboostjob.Name)
	r.Log.Info("job deleted", "namespace", xgboostjob.Namespace, "name", xgboostjob.Name)
	return nil
}

// GetJobFromInformerCache returns the Job from Informer Cache
func (r *XGBoostJobReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	job := &xgboostv1.XGBoostJob{}
	// Default reader for XGBoostJob is cache reader.
	err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			r.Log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// GetJobFromAPIClient returns the Job from API server
func (r *XGBoostJobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	job := &xgboostv1.XGBoostJob{}

	// TODO (Jeffwan@): consider to read from apiserver directly.
	err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "xgboost job not found", "namespace", namespace, "name", name)
		} else {
			r.Log.Error(err, "failed to get job from api-server", "namespace", namespace, "name", name)
		}
		return nil, err
	}
	return job, nil
}

// UpdateJobStatus updates the job status and job conditions
func (r *XGBoostJobReconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	xgboostJob, ok := job.(*xgboostv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of xgboostJob", xgboostJob)
	}

	for rtype, spec := range replicas {
		status := jobStatus.ReplicaStatuses[rtype]

		succeeded := status.Succeeded
		expected := *(spec.Replicas) - succeeded
		running := status.Active
		failed := status.Failed

		logrus.Infof("XGBoostJob=%s, ReplicaType=%s expected=%d, running=%d, succeeded=%d , failed=%d",
			xgboostJob.Name, rtype, expected, running, succeeded, failed)

		if rtype == commonv1.ReplicaType(xgboostv1.XGBoostReplicaTypeMaster) {
			if running > 0 {
				msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, xgboostJobRunningReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
			}
			// when master is succeed, the job is finished.
			if expected == 0 {
				msg := fmt.Sprintf("XGBoostJob %s is successfully completed.", xgboostJob.Name)
				logrus.Info(msg)
				r.Recorder.Event(xgboostJob, k8sv1.EventTypeNormal, xgboostJobSucceededReason, msg)
				if jobStatus.CompletionTime == nil {
					now := metav1.Now()
					jobStatus.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobSucceeded, xgboostJobSucceededReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
				return nil
			}
		}
		if failed > 0 {
			if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				msg := fmt.Sprintf("XGBoostJob %s is restarting because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.Recorder.Event(xgboostJob, k8sv1.EventTypeWarning, xgboostJobRestartingReason, msg)
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRestarting, xgboostJobRestartingReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
			} else {
				msg := fmt.Sprintf("XGBoostJob %s is failed because %d %s replica(s) failed.", xgboostJob.Name, failed, rtype)
				r.Recorder.Event(xgboostJob, k8sv1.EventTypeNormal, xgboostJobFailedReason, msg)
				if xgboostJob.Status.CompletionTime == nil {
					now := metav1.Now()
					xgboostJob.Status.CompletionTime = &now
				}
				err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobFailed, xgboostJobFailedReason, msg)
				if err != nil {
					logger.LoggerForJob(xgboostJob).Infof("Append job condition error: %v", err)
					return err
				}
			}
		}
	}

	// Some workers are still running, leave a running condition.
	msg := fmt.Sprintf("XGBoostJob %s is running.", xgboostJob.Name)
	logger.LoggerForJob(xgboostJob).Infof(msg)

	if err := commonutil.UpdateJobConditions(jobStatus, commonv1.JobRunning, xgboostJobRunningReason, msg); err != nil {
		logger.LoggerForJob(xgboostJob).Error(err, "failed to update XGBoost Job conditions")
		return err
	}

	return nil
}

// UpdateJobStatusInApiServer updates the job status in to cluster.
func (r *XGBoostJobReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	xgboostjob, ok := job.(*xgboostv1.XGBoostJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of XGBoostJob", xgboostjob)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&xgboostjob.Status.JobStatus, jobStatus) {
		xgboostjob = xgboostjob.DeepCopy()
		xgboostjob.Status.JobStatus = *jobStatus.DeepCopy()
	}

	result := r.Update(context.Background(), xgboostjob)

	if result != nil {
		logger.LoggerForJob(xgboostjob).Error(result, "failed to update XGBoost Job conditions in the API server")
		return result
	}

	return nil
}

// onOwnerCreateFunc modify creation condition.
func onOwnerCreateFunc(r reconcile.Reconciler) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		xgboostJob, ok := e.Object.(*xgboostv1.XGBoostJob)
		if !ok {
			return true
		}
		scheme.Scheme.Default(xgboostJob)
		msg := fmt.Sprintf("xgboostJob %s is created.", e.Object.GetName())
		logrus.Info(msg)
		//specific the run policy

		if xgboostJob.Spec.RunPolicy.CleanPodPolicy == nil {
			xgboostJob.Spec.RunPolicy.CleanPodPolicy = new(commonv1.CleanPodPolicy)
			xgboostJob.Spec.RunPolicy.CleanPodPolicy = &defaultCleanPodPolicy
		}

		if err := commonutil.UpdateJobConditions(&xgboostJob.Status.JobStatus, commonv1.JobCreated, xgboostJobCreatedReason, msg); err != nil {
			log.Log.Error(err, "append job condition error")
			return false
		}
		return true
	}
}
